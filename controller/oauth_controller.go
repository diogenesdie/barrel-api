package controller

import (
	"barrel-api/model"
	"barrel-api/repository"
	"barrel-api/token"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"
)

// Domínios de redirect aceitos pela Amazon para account linking.
var alexaAllowedRedirectPrefixes = []string{
	"https://pitangui.amazon.com",  // NA
	"https://layla.amazon.com",     // EU
	"https://alexa.amazon.co.jp",   // FE
}

var loginTemplate = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Barrel — Vincular conta Alexa</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: #0A0A0A;
      color: #fff;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
    }
    .card {
      background: #1A1A1A;
      padding: 2rem;
      border-radius: 16px;
      width: 340px;
      box-shadow: 0 8px 32px rgba(0,0,0,0.5);
    }
    .logo {
      text-align: center;
      font-size: 1.75rem;
      font-weight: 700;
      color: #3B82F6;
      margin-bottom: 0.25rem;
    }
    .subtitle {
      text-align: center;
      font-size: 0.875rem;
      color: #6B7280;
      margin-bottom: 1.75rem;
    }
    label {
      display: block;
      font-size: 0.875rem;
      color: #9CA3AF;
      margin-bottom: 0.375rem;
    }
    input[type=text], input[type=password] {
      width: 100%;
      padding: 0.75rem 1rem;
      margin-bottom: 1rem;
      border-radius: 8px;
      border: 1px solid #374151;
      background: #111827;
      color: #fff;
      font-size: 1rem;
    }
    input:focus { outline: none; border-color: #3B82F6; }
    button {
      width: 100%;
      padding: 0.875rem;
      background: #3B82F6;
      color: #fff;
      border: none;
      border-radius: 8px;
      cursor: pointer;
      font-size: 1rem;
      font-weight: 600;
    }
    button:hover { background: #2563EB; }
    .error {
      background: rgba(239,68,68,0.15);
      border: 1px solid rgba(239,68,68,0.3);
      color: #FCA5A5;
      padding: 0.75rem;
      border-radius: 8px;
      font-size: 0.875rem;
      margin-bottom: 1rem;
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="logo">Barrel</div>
    <p class="subtitle">Vinculação com Amazon Alexa</p>
    {{if .ErrorMsg}}<div class="error">{{.ErrorMsg}}</div>{{end}}
    <form method="POST">
      <input type="hidden" name="redirect_uri" value="{{.RedirectURI}}">
      <input type="hidden" name="state"        value="{{.State}}">
      <label for="username">Usuário</label>
      <input type="text"     id="username" name="username" placeholder="Digite seu usuário"
             required autocomplete="username" value="{{.Username}}">
      <label for="password">Senha</label>
      <input type="password" id="password" name="password" placeholder="Digite sua senha"
             required autocomplete="current-password">
      <button type="submit">Vincular conta</button>
    </form>
  </div>
</body>
</html>`))

type OAuthController struct {
	oauthRepo   *repository.OAuthRepository
	sessionRepo *repository.SessionRepository
	db          *sql.DB
}

func NewOAuthController(oauthRepo *repository.OAuthRepository, sessionRepo *repository.SessionRepository, db *sql.DB) *OAuthController {
	return &OAuthController{
		oauthRepo:   oauthRepo,
		sessionRepo: sessionRepo,
		db:          db,
	}
}

// AuthorizeHandler handles GET /auth/v1/oauth/authorize (mostra formulário)
// e POST /auth/v1/oauth/authorize (processa login e redireciona com code).
func (oc *OAuthController) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		oc.showLoginForm(w, r, "", "")
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	redirectURI := r.FormValue("redirect_uri")
	state := r.FormValue("state")
	username := r.FormValue("username")
	password := r.FormValue("password")

	if !isAllowedRedirectURI(redirectURI) {
		http.Error(w, "redirect_uri não autorizado", http.StatusBadRequest)
		return
	}

	// Autentica o usuário reutilizando a lógica do session_repository
	userID, err := oc.authenticateUser(username, password)
	if err != nil {
		oc.showLoginFormWithError(w, redirectURI, state, username, "Usuário ou senha incorretos. Tente novamente.")
		return
	}

	// Gera código de autorização aleatório (32 bytes → 64 chars hex)
	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	codeStr := hex.EncodeToString(codeBytes)

	authCode := &model.OAuthCode{
		Code:        codeStr,
		UserID:      userID,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	if err := oc.oauthRepo.CreateCode(authCode); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, redirectURI+"?code="+codeStr+"&state="+state, http.StatusFound)
}

// TokenHandler handles POST /auth/v1/oauth/token
// Troca o authorization code por um JWT access_token de 30 dias.
func (oc *OAuthController) TokenHandler(w http.ResponseWriter, r *http.Request) {
	var grantType, code, redirectURI string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			writeResponse(w, http.StatusBadRequest, "invalid form body", nil)
			return
		}
		grantType = r.FormValue("grant_type")
		code = r.FormValue("code")
		redirectURI = r.FormValue("redirect_uri")
	} else {
		var body struct {
			GrantType   string `json:"grant_type"`
			Code        string `json:"code"`
			RedirectURI string `json:"redirect_uri"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeResponse(w, http.StatusBadRequest, "invalid JSON body", nil)
			return
		}
		grantType = body.GrantType
		code = body.Code
		redirectURI = body.RedirectURI
	}

	if grantType != "authorization_code" {
		writeResponse(w, http.StatusBadRequest, "unsupported grant_type", nil)
		return
	}
	if code == "" || redirectURI == "" {
		writeResponse(w, http.StatusBadRequest, "code e redirect_uri são obrigatórios", nil)
		return
	}

	oauthCode, err := oc.oauthRepo.ConsumeCode(code, redirectURI)
	if err != nil {
		switch err {
		case repository.ErrOAuthCodeNotFound:
			writeResponse(w, http.StatusBadRequest, "código inválido", nil)
		case repository.ErrOAuthCodeExpired:
			writeResponse(w, http.StatusBadRequest, "código expirado", nil)
		case repository.ErrOAuthCodeUsed:
			writeResponse(w, http.StatusBadRequest, "código já utilizado", nil)
		default:
			writeResponse(w, http.StatusInternalServerError, "erro interno", nil)
		}
		return
	}

	// Token de 30 dias para não exigir re-vinculação frequente
	const tokenDuration = 30 * 24 * time.Hour
	tokenString, err := token.GenerateTokenWithDuration(oauthCode.UserID, tokenDuration)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "erro ao gerar token", nil)
		return
	}

	// Insere session row para que o AuthenticationMiddleware valide o token
	now := time.Now()
	_, err = oc.db.Exec(`
		INSERT INTO barrel.sessions (id, user_id, token, status, expires_at, created_at, updated_at)
		VALUES (nextval('barrel.seq_sessions'), $1, $2, 'A', $3, $4, $4)
	`, oauthCode.UserID, tokenString, now.Add(tokenDuration), now)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, "erro ao registrar sessão", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   int(tokenDuration.Seconds()),
	})
}

// authenticateUser valida username/password contra o banco e retorna o userID.
func (oc *OAuthController) authenticateUser(username, password string) (uint64, error) {
	var userID uint64
	var passwordMatch bool
	err := oc.db.QueryRow(`
		SELECT u.id,
		       CASE WHEN crypt($2::text, u.password) = u.password THEN true ELSE false END
		  FROM barrel.users u
		 WHERE u.username = $1
		   AND u.status   = 'A'
	`, username, password).Scan(&userID, &passwordMatch)

	if err != nil {
		return 0, err
	}
	if !passwordMatch {
		return 0, repository.ErrInvalidPassword
	}
	return userID, nil
}

func (oc *OAuthController) showLoginForm(w http.ResponseWriter, r *http.Request, errMsg, username string) {
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	if !isAllowedRedirectURI(redirectURI) {
		http.Error(w, "redirect_uri não autorizado", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTemplate.Execute(w, map[string]string{
		"RedirectURI": redirectURI,
		"State":       state,
		"ErrorMsg":    errMsg,
		"Username":    username,
	})
}

func (oc *OAuthController) showLoginFormWithError(w http.ResponseWriter, redirectURI, state, username, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	loginTemplate.Execute(w, map[string]string{
		"RedirectURI": redirectURI,
		"State":       state,
		"ErrorMsg":    errMsg,
		"Username":    username,
	})
}

func isAllowedRedirectURI(uri string) bool {
	if uri == "" {
		return false
	}
	for _, prefix := range alexaAllowedRedirectPrefixes {
		if strings.HasPrefix(uri, prefix) {
			return true
		}
	}
	return false
}
