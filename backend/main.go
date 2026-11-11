package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// The prototype stores accounts and balances in a local JSON file. Set
// MEGACITY_SECRET to a private random value before exposing this service.
type Account struct {
	ID string `json:"id"`
	Nickname string `json:"nickname"`
	Email string `json:"email"`
	PasswordHash string `json:"-"`
	Salt string `json:"-"`
	Coins int `json:"coins"`
	JoinedAt time.Time `json:"joinedAt"`
	Purchases []string `json:"purchases"`
}
type database struct { Accounts map[string]Account `json:"accounts"` }
type Server struct { mu sync.Mutex; db database; path string; secret []byte }
type Product struct { ID string `json:"id"`; Name string `json:"name"`; Description string `json:"description"`; Price int `json:"price"`; Icon string `json:"icon"`; Category string `json:"category"` }
var products = []Product{
	{ "neon-car", "Неоновый спорткар", "Эксклюзивный скин для MegaCity Car", 1200, "🏎️", "Транспорт" },
	{ "penthouse", "Квартира с видом", "Премиальные апартаменты в центре города", 900, "🏙️", "Недвижимость" },
	{ "city-pass", "Пропуск жителя", "Особый значок и доступ к городским событиям", 500, "🎟️", "Привилегии" },
	{ "golden-horn", "Золотой сигнал", "Редкий сигнал для вашего автомобиля", 350, "📯", "Транспорт" },
	{ "sky-apartment", "Небесная квартира", "Уютный пентхаус над облаками", 1800, "🌇", "Недвижимость" },
	{ "founder-badge", "Значок первопроходца", "Коллекционный значок MegaCity", 250, "✨", "Коллекция" },
}
func main() {
	secret := os.Getenv("MEGACITY_SECRET"); if secret == "" { secret = "local-development-secret-change-before-deploy" }
	s := &Server{path: env("MEGACITY_DATA", "data/megacity.json"), secret: []byte(secret), db: database{Accounts: map[string]Account{}}}
	if err := s.load(); err != nil { log.Fatal(err) }
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("GET /api/me", s.me)
	mux.HandleFunc("GET /api/shop/products", s.catalog)
	mux.HandleFunc("POST /api/shop/buy", s.buy)
	addr := env("MEGACITY_ADDR", ":8080")
	log.Printf("MegaCity API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}
func env(k, fallback string) string { if v := os.Getenv(k); v != "" { return v }; return fallback }
func (s *Server) load() error { b, err := os.ReadFile(s.path); if errors.Is(err, os.ErrNotExist) { return nil }; if err != nil { return err }; return json.Unmarshal(b, &s.db) }
func (s *Server) save() error { if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil { return err }; b, err := json.MarshalIndent(s.db, "", "  "); if err != nil { return err }; tmp := s.path+".tmp"; if err=os.WriteFile(tmp,b,0600); err != nil { return err }; return os.Rename(tmp,s.path) }
func (s *Server) health(w http.ResponseWriter,r *http.Request) { respond(w,200,map[string]string{"status":"online","service":"MegaCity API"}) }
func (s *Server) register(w http.ResponseWriter,r *http.Request) { var in struct{Nickname string `json:"nickname"`; Email string `json:"email"`; Password string `json:"password"`}; if !decode(w,r,&in) {return}; in.Nickname=strings.TrimSpace(in.Nickname); in.Email=strings.ToLower(strings.TrimSpace(in.Email)); if len(in.Nickname)<3||len(in.Nickname)>20||len(in.Password)<8||!strings.Contains(in.Email,"@") {fail(w,400,"Укажите ник от 3 до 20 символов, корректную почту и пароль от 8 символов.");return}; s.mu.Lock(); defer s.mu.Unlock(); for _,a:=range s.db.Accounts { if strings.EqualFold(a.Nickname,in.Nickname)||a.Email==in.Email {fail(w,409,"Никнейм или почта уже зарегистрированы.");return} }; id:=random(16); salt:=random(16); a:=Account{ID:id,Nickname:in.Nickname,Email:in.Email,Salt:salt,PasswordHash:passwordHash(in.Password,salt),Coins:1000,JoinedAt:time.Now().UTC(),Purchases:[]string{}}; s.db.Accounts[id]=a; if err:=s.save();err!=nil {fail(w,500,"Не удалось сохранить аккаунт.");return}; s.session(w,a) }
func (s *Server) login(w http.ResponseWriter,r *http.Request) { var in struct{Login string `json:"login"`; Password string `json:"password"`}; if !decode(w,r,&in){return}; s.mu.Lock(); defer s.mu.Unlock(); for _,a:=range s.db.Accounts { if strings.EqualFold(a.Nickname,in.Login)||a.Email==strings.ToLower(in.Login) { if hmac.Equal([]byte(a.PasswordHash),[]byte(passwordHash(in.Password,a.Salt))) {s.session(w,a);return}; break } }; fail(w,401,"Неверный никнейм, почта или пароль.") }
func (s *Server) me(w http.ResponseWriter,r *http.Request) { a,ok:=s.authorized(r); if !ok {fail(w,401,"Войдите в аккаунт.");return}; respond(w,200,map[string]any{"player":public(a)}) }
func (s *Server) catalog(w http.ResponseWriter,r *http.Request) { respond(w,200,map[string]any{"products":products,"currency":"MC","starterBalance":1000}) }
func (s *Server) buy(w http.ResponseWriter,r *http.Request) { a,ok:=s.authorized(r); if !ok {fail(w,401,"Войдите в аккаунт, чтобы совершить покупку.");return}; var in struct{ProductID string `json:"productId"`}; if !decode(w,r,&in){return}; var p *Product; for i:=range products {if products[i].ID==in.ProductID {p=&products[i];break}}; if p==nil {fail(w,404,"Товар не найден.");return}; s.mu.Lock(); defer s.mu.Unlock(); a=s.db.Accounts[a.ID]; for _,id:=range a.Purchases {if id==p.ID {fail(w,409,"Этот товар уже куплен.");return}}; if a.Coins<p.Price {fail(w,400,"Недостаточно игровой валюты.");return}; a.Coins-=p.Price; a.Purchases=append(a.Purchases,p.ID); s.db.Accounts[a.ID]=a; if err:=s.save();err!=nil {fail(w,500,"Не удалось завершить покупку.");return}; respond(w,200,map[string]any{"player":public(a),"product":p,"message":"Покупка успешно завершена!"}) }
func (s *Server) session(w http.ResponseWriter,a Account) { payload:=base64.RawURLEncoding.EncodeToString([]byte(a.ID+"|"+time.Now().Add(24*time.Hour).Format(time.RFC3339))); mac:=hmac.New(sha256.New,s.secret); mac.Write([]byte(payload)); token:=payload+"."+base64.RawURLEncoding.EncodeToString(mac.Sum(nil)); respond(w,200,map[string]any{"token":token,"player":public(a)}) }
func public(a Account) map[string]any {return map[string]any{"id":a.ID,"nickname":a.Nickname,"email":a.Email,"coins":a.Coins,"joinedAt":a.JoinedAt,"purchases":a.Purchases}}
func (s *Server) authorized(r *http.Request)(Account,bool) { h:=strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer "); parts:=strings.Split(h,"."); if len(parts)!=2{return Account{},false}; mac:=hmac.New(sha256.New,s.secret);mac.Write([]byte(parts[0])); sig,err:=base64.RawURLEncoding.DecodeString(parts[1]);if err!=nil||!hmac.Equal(sig,mac.Sum(nil)){return Account{},false}; raw,err:=base64.RawURLEncoding.DecodeString(parts[0]);if err!=nil{return Account{},false}; fields:=strings.Split(string(raw),"|");if len(fields)!=2{return Account{},false}; exp,err:=time.Parse(time.RFC3339,fields[1]);if err!=nil||time.Now().After(exp){return Account{},false};s.mu.Lock();defer s.mu.Unlock();a,ok:=s.db.Accounts[fields[0]];return a,ok }
func passwordHash(password,salt string) string { // PBKDF2-HMAC-SHA256, one block, 120k iterations.
	mac:=hmac.New(sha256.New,[]byte(password));mac.Write([]byte(salt));mac.Write([]byte{0,0,0,1});u:=mac.Sum(nil);out:=append([]byte(nil),u...);for i:=1;i<120000;i++{mac=hmac.New(sha256.New,[]byte(password));mac.Write(u);u=mac.Sum(nil);for j:=range out{out[j]^=u[j]}};return base64.RawStdEncoding.EncodeToString(out)
}
func random(n int) string {b:=make([]byte,n);if _,err:=rand.Read(b);err!=nil {panic(err)};return base64.RawURLEncoding.EncodeToString(b)}
func decode(w http.ResponseWriter,r *http.Request,v any)bool {r.Body=http.MaxBytesReader(w,r.Body,1<<20);d:=json.NewDecoder(r.Body);if err:=d.Decode(v);err!=nil{fail(w,400,"Некорректный запрос.");return false};return true}
func respond(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json; charset=utf-8");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func fail(w http.ResponseWriter,status int,message string){respond(w,status,map[string]string{"error":message})}
func cors(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){origin:=r.Header.Get("Origin");if strings.HasPrefix(origin,"http://localhost:")||strings.HasPrefix(origin,"http://127.0.0.1:"){w.Header().Set("Access-Control-Allow-Origin",origin);w.Header().Set("Vary","Origin")};w.Header().Set("Access-Control-Allow-Headers","Authorization, Content-Type");w.Header().Set("Access-Control-Allow-Methods","GET, POST, OPTIONS");if r.Method==http.MethodOptions{w.WriteHeader(http.StatusNoContent);return};next.ServeHTTP(w,r)})}
