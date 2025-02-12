package lnurl

import (
	"encoding/json"
	"github.com/LightningTipBot/LightningTipBot/internal"
	"github.com/LightningTipBot/LightningTipBot/internal/telegram"
	"net/http"
	"net/url"
	"time"

	"github.com/LightningTipBot/LightningTipBot/internal/lnbits"
	"github.com/LightningTipBot/LightningTipBot/internal/storage"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Server struct {
	httpServer       *http.Server
	bot              *telegram.TipBot
	c                *lnbits.Client
	database         *gorm.DB
	callbackHostname *url.URL
	buntdb           *storage.DB
	WebhookServer    string
}

const (
	statusError    = "ERROR"
	statusOk       = "OK"
	payRequestTag  = "payRequest"
	lnurlEndpoint  = ".well-known/lnurlp"
	minSendable    = 1000 // mSat
	MaxSendable    = 1_000_000_000
	CommentAllowed = 256
)

func NewServer(bot *telegram.TipBot) *Server {
	srv := &http.Server{
		Addr: internal.Configuration.Bot.LNURLServerUrl.Host,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	apiServer := &Server{
		c:                bot.Client,
		database:         bot.Database,
		bot:              bot,
		httpServer:       srv,
		callbackHostname: internal.Configuration.Bot.LNURLHostUrl,
		WebhookServer:    internal.Configuration.Lnbits.WebhookServer,
		buntdb:           bot.Bunt,
	}

	apiServer.httpServer.Handler = apiServer.newRouter()
	go apiServer.httpServer.ListenAndServe()
	log.Infof("[LNURL] Server started at %s", internal.Configuration.Bot.LNURLServerUrl.Host)
	return apiServer
}

func (w *Server) newRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/.well-known/lnurlp/{username}", w.handleLnUrl).Methods(http.MethodGet)
	router.HandleFunc("/@{username}", w.handleLnUrl).Methods(http.MethodGet)
	return router
}

func NotFoundHandler(writer http.ResponseWriter, err error) {
	log.Errorln(err)
	// return 404 on any error
	http.Error(writer, "404 page not found", http.StatusNotFound)
}

func writeResponse(writer http.ResponseWriter, response interface{}) error {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = writer.Write(jsonResponse)
	return err
}
