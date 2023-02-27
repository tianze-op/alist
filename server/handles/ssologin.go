package handles

import (
	"errors"
	"net/url"

	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

func SSOLoginRedirect(c *gin.Context) {
	method := c.Query("method")
	enabled, err := db.GetSettingItemByKey("sso_login_enabled")
	clientId, err := db.GetSettingItemByKey("sso_client_id")
	platform, err := db.GetSettingItemByKey("sso_login_platform")
	var r_url string
	var redirect_uri string
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	} else if enabled.Value == "true" {
		urlValues := url.Values{}
		if method == "" {
			common.ErrorStrResp(c, "no method provided", 400)
			return
		}
		redirect_uri = common.GetApiUrl(c.Request) + "/api/auth/sso_callback" + "?method=" + method
		urlValues.Add("response_type", "code")
		urlValues.Add("redirect_uri", redirect_uri)
		urlValues.Add("client_id", clientId.Value)
		switch platform.Value {
		case "Github":
			r_url = "https://github.com/login/oauth/authorize?"
			urlValues.Add("scope", "read:user")
		case "Microsoft":
			r_url = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize?"
			urlValues.Add("scope", "user.read")
			urlValues.Add("response_mode", "query")
		case "Google":
			r_url = "https://accounts.google.com/o/oauth2/v2/auth?"
			urlValues.Add("scope", "https://www.googleapis.com/auth/userinfo.profile")
		case "Dingtalk":
			r_url = "https://login.dingtalk.com/oauth2/auth?"
			urlValues.Add("scope", "openid")
			urlValues.Add("prompt", "consent")
			urlValues.Add("response_type", "code")
		default:
			common.ErrorStrResp(c, "invalid platform", 400)
			return
		}
		c.Redirect(302, r_url+urlValues.Encode())
	} else {
		common.ErrorStrResp(c, "Single sign-on is not enabled", 403)
	}
}

var ssoClient = resty.New().SetRetryCount(3)

func SSOLoginCallback(c *gin.Context) {
	argument := c.Query("method")
	if argument == "get_sso_id" || argument == "sso_get_token" {
		enabled, err := db.GetSettingItemByKey("sso_login_enabled")
		clientId, err := db.GetSettingItemByKey("sso_client_id")
		clientSecret, err := db.GetSettingItemByKey("sso_client_secrets")
		platform, err := db.GetSettingItemByKey("sso_login_platform")
		var url1, url2, additionalbody, scope, authstring, idstring string
		switch platform.Value {
		case "Github":
			url1 = "https://github.com/login/oauth/access_token"
			url2 = "https://api.github.com/user"
			additionalbody = ""
			authstring = "code"
			scope = "read:user"
			idstring = "id"
			break
		case "Microsoft":
			url1 = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
			url2 = "https://graph.microsoft.com/v1.0/me"
			additionalbody = "&grant_type=authorization_code"
			scope = "user.read"
			authstring = "code"
			idstring = "id"
			break
		case "Google":
			url1 = "https://oauth2.googleapis.com/token"
			url2 = "https://www.googleapis.com/oauth2/v1/userinfo"
			additionalbody = "&grant_type=authorization_code"
			scope = "https://www.googleapis.com/auth/userinfo.profile"
			authstring = "code"
			idstring = "id"
			break
		case "Dingtalk":
			url1 = "https://api.dingtalk.com/v1.0/oauth2/userAccessToken"
			url2 = "https://api.dingtalk.com/v1.0/contact/users/me"
			authstring = "authCode"
			idstring = "unionId"
			break
		default:
			common.ErrorStrResp(c, "invalid platform", 400)
			return
		}
		if err != nil {
			common.ErrorResp(c, err, 400)
			return
		} else if enabled.Value == "true" {
			callbackCode := c.Query(authstring)
			if callbackCode == "" {
				common.ErrorStrResp(c, "No code provided", 400)
				return
			}
			var resp *resty.Response
			var err error
			if platform.Value == "Dingtalk" {
				resp, err = ssoClient.R().SetHeader("content-type", "application/json").SetHeader("Accept", "application/json").
					SetBody(map[string]string{
						"clientId":     clientId.Value,
						"clientSecret": clientSecret.Value,
						"code":         callbackCode,
						"grantType":    "authorization_code",
					}).
					Post(url1)
			} else {
				resp, err = ssoClient.R().SetHeader("content-type", "application/x-www-form-urlencoded").SetHeader("Accept", "application/json").
					SetBody("client_id=" + clientId.Value + "&client_secret=" + clientSecret.Value + "&code=" + callbackCode + "&redirect_uri=" + common.GetApiUrl(c.Request) + "/api/auth/sso_callback?method=" + argument + "&scope=" + scope + additionalbody).
					Post(url1)
			}
			if err != nil {
				common.ErrorResp(c, err, 400)
				return
			}
			if platform.Value == "Dingtalk" {
				accessToken := utils.Json.Get(resp.Body(), "accessToken").ToString()
				resp, err = ssoClient.R().SetHeader("x-acs-dingtalk-access-token", accessToken).
					Get(url2)
			} else {
				accessToken := utils.Json.Get(resp.Body(), "access_token").ToString()
				resp, err = ssoClient.R().SetHeader("Authorization", "Bearer "+accessToken).
					Get(url2)
			}
			if err != nil {
				common.ErrorResp(c, err, 400)
				return
			}
			UserID := utils.Json.Get(resp.Body(), idstring).ToString()
			if UserID == "0" {
				common.ErrorResp(c, errors.New("error occured"), 400)
				return
			}
			if argument == "get_sso_id" {
				html := `<!DOCTYPE html>
				<head></head>
				<body>
				<script>
				window.opener.postMessage("` + UserID + `", "*")
				window.close()
				</script>
				</body>`
				c.Data(200, "text/html; charset=utf-8", []byte(html))
				return
			}
			if argument == "sso_get_token" {
				user, err := db.GetUserBySSOID(UserID)
				if err != nil {
					common.ErrorResp(c, err, 400)
				}
				token, err := common.GenerateToken(user.Username)
				if err != nil {
					common.ErrorResp(c, err, 400)
				}
				html := `<!DOCTYPE html>
				<head></head>
				<body>
				<script>
				window.opener.postMessage("` + token + `", "*")
				window.close()
				</script>
				</body>`
				c.Data(200, "text/html; charset=utf-8", []byte(html))
				return
			}
		} else {
			common.ErrorResp(c, errors.New("invalid request"), 500)
		}
	}
}
