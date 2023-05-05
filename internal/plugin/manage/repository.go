package plugin_manage

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/setting"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func init() {
	op.RegisterSettingItemHook(conf.PluginRepository, func(item *model.SettingItem) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		if err := UpdatePluginRepository(ctx); err != nil {
			utils.Log.Warnf("UpdatePluginRepository err: %s", err)
		}
		return nil
	})
}

var pluginRepository = map[string]model.PluginInfo{}

// 更新插件仓库
func UpdatePluginRepository(ctx context.Context) error {
	newPluginRepository := map[string]model.PluginInfo{}
	var errs []error

	// download repository
	repositorys := strings.Split(setting.GetStr(conf.PluginRepository), "\n")
	for _, repositoryUrl := range repositorys {
		if repositoryUrl == "" {
			continue
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, repositoryUrl, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errs = append(errs, errors.Errorf("repository:%s,err: %s", repositoryUrl, err))
			continue
		}

		var repository model.PluginRepository
		if err := utils.Json.NewDecoder(resp.Body).Decode(&repository); err != nil {
			errs = append(errs, errors.Errorf("repository:%s, httpStatusCode:%d, err: %s", repositoryUrl, resp.StatusCode, err))
			continue
		}
		for _, plugin := range repository.Plugins {
			newPluginRepository[plugin.UUID] = plugin
		}
	}

	pluginRepository = newPluginRepository
	return utils.MergeErrors(errs...)
}

// 获取所有插件
func GetAllPluginRepository() []model.PluginInfo {
	plugins := make([]model.PluginInfo, 0, len(pluginRepository))
	for _, plugin := range pluginRepository {
		plugins = append(plugins, plugin)
	}
	return plugins
}

func GetPluginRepository(uuid string) (model.PluginInfo, bool) {
	p, ok := pluginRepository[uuid]
	return p, ok
}

// 检测是否存在更新
func CheckUpdate(uuid string, oldVersion string) (has bool, found bool) {
	if plugin, found := pluginRepository[uuid]; found {
		for _, downInfo := range plugin.Downloads {
			if CompareVersionStr(downInfo.Version, oldVersion) == VersionBig {
				if IsSupportPlugin(strings.Join(downInfo.ApiVersion, ",")) {
					return true, true
				}
			}
		}
		return false, true
	}
	return false, false
}

// 下载插件
func DownloadPlugin(ctx context.Context, uuid string, version string, install func(*os.File, model.PluginInfo) error) error {
	if plugin, found := pluginRepository[uuid]; found {
		for _, downInfo := range plugin.Downloads {
			if downInfo.Version == version {
				// get
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, downInfo.DownloadUrl, nil)
				if err != nil {
					return err
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return err
				}
				// save
				file, err := utils.CreateTempFile(resp.Body)
				if err != nil {
					return err
				}

				defer os.Remove(file.Name())
				defer file.Close()

				// install
				if install != nil {
					return install(file, plugin)
				}
				return nil
			}
		}
		return errs.NotFoundPluginVersionByRepository
	}
	return errs.NotFoundPluginByRepository
}

// 安装插件并写入数据库
func InstallPlugin(ctx context.Context, uuid string, version string) (*model.Plugin, error) {
	if _, err := db.GetPluginByUUID(uuid); err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.PluginHasBeenInstalled
	}

	var (
		path string
		mode string
	)

	err := DownloadPlugin(ctx, uuid, version, func(f *os.File, plugin model.PluginInfo) error {
		switch plugin.Mode {
		case model.PLUGIN_MODE_YAEGI:
			if err := UnzipArchive(f, filepath.Join(conf.Conf.PluginDir, "src", plugin.UUID)); err != nil {
				return err
			}
			path = plugin.UUID
			mode = model.PLUGIN_MODE_YAEGI
			return nil
		}
		return errs.NotSupportPluginMode
	})
	if err != nil {
		return nil, err
	}
	return FixPluginConfigByModel(model.Plugin{Mode: mode, Path: path, Disabled: false})
}

// 删除插件文件并移除数据库
func UninstallPlugin(plugin model.Plugin) error {
	switch plugin.Mode {
	case model.PLUGIN_MODE_YAEGI:
		if err := os.RemoveAll(filepath.Join(conf.Conf.PluginDir, "src", plugin.Path)); err != nil {
			return err
		}
	}
	return db.DeletePluginByID(plugin.ID)
}
