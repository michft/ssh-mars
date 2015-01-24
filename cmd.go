package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"github.com/duncankl/zbase32"
	"github.com/jessevdk/go-flags"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"os/signal"
	"strings"
	"time"
)

type Options struct {
	SSHKey        string `long:"ssh-key" description:"Private key to identify server with." default:"ssh-identity"`
	TLSCert       string `long:"tls-cert" description:"TLS certificate file." default:"tls-identity.crt"`
	TLSKey        string `long:"tls-key" description:"TLS private key file." default:"tls-identity.key"`
	Database      string `long:"db" description:"SQLite database used to store persistent data." default:"mars.sqlite"`
	SSHListen     string `long:"ssh-listen" description:"Host and port for SSH server to listen on." default:":2022"`
	SSHAdvertise  string `long:"ssh-advertise" description:"Host and port of SSH server as visible to users." default:"localhost:2022"`
	HTTPListen    string `long:"http-listen" description:"Host and port for HTTP server to listen on." default:":3000"`
	HTTPAdvertise string `long:"http-advertise" description:"Host and port of HTTP server as visible to users." default:"localhost:3000"`
	AssetsDir     string `long:"assets" description:"Directory containing the web assets." default:"assets"`
	Log           string `long:"log" description:"Log file for HTTP requests." default:""`
}

const (
	signinIdLength      = 4
	signinSecretLength  = 8
	sessionIdLength     = 8
	sessionSecretLength = 40
	csrfTokenLength     = 40
)

func main() {
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	var logger *lumberjack.Logger

	if options.Log != "" {
		logger = &lumberjack.Logger{
			Filename:   options.Log,
			MaxSize:    1,
			MaxBackups: 3,
			MaxAge:     28,
		}
	}

	db, err := setupDB(options.Database)
	if err != nil {
		fmt.Fprintln(os.Stderr, "setting up database:", err)
		os.Exit(1)
	}

	hostKey, err := readPrivateKey(options.SSHKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	hostPubkey := hostKey.PublicKey()

	broker := NewBroker()

	generateSigninURL := func(pubkey []byte, token string) error {
		return authenticateSigninRequest(db, broker, pubkey, token)
	}

	err = startSSHServer(options.SSHListen, hostKey, generateSigninURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "starting SSH server:", err)
		os.Exit(1)
	}

	// TODO: handle errors binding to port
	startWebServer(options.HTTPListen, options.TLSCert, options.TLSKey, options.SSHAdvertise, options.AssetsDir, logger, hostPubkey, db, broker)

	sessionSweeper(db)

	fmt.Println("Server started.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func authenticateSigninRequest(db *sql.DB, broker *Broker, providedPubkey []byte, signinToken string) error {
	if len(providedPubkey) > 10000 {
		return fmt.Errorf("public key is too large (%v bytes)", len(providedPubkey))
	}

	if len(signinToken) != signinIdLength+signinSecretLength {
		return fmt.Errorf("invalid length for signin token: %v", len(signinToken))
	}

	signinId := signinToken[:signinIdLength]
	providedSigninSecret := signinToken[signinIdLength:]

	var signinSecret string
	var pubkey []byte
	err := db.QueryRow("select signin_secret, pubkey from signin_requests where signin_id = ?", signinId).Scan(&signinSecret, &pubkey)

	if err == sql.ErrNoRows {
		return fmt.Errorf("no signin request for token: %q", signinToken)
	} else if err != nil {
		return fmt.Errorf("retrieving signin token:", err)
	}

	if subtle.ConstantTimeCompare([]byte(providedSigninSecret), []byte(signinSecret)) != 1 {
		return fmt.Errorf("incorrect signin token: %q", signinToken)
	}

	if len(pubkey) != 0 && subtle.ConstantTimeCompare(providedPubkey, pubkey) != 1 {
		return fmt.Errorf("this signin request has already been authenticated")
	}

	_, err = db.Exec("update signin_requests set pubkey = ? where signin_id = ? and signin_secret = ?", providedPubkey, signinId, signinSecret)
	if err != nil {
		return err
	}

	broker.Notifier <- signinToken

	return nil
}

func sessionSweeper(db *sql.DB) {
	c := time.Tick(1 * time.Minute)
	go func() {
		for t := range c {
			db.Exec("delete from signin_requests where created_at < ?", t.Unix()-(60*10))
			db.Exec("delete from sessions where last_active < ?", t.Unix()-(3600*24*20))
		}
	}()
}

func randomToken(chars int) (string, error) {
	bits := chars * 5
	randomBytes := make([]byte, (bits+7)/8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return zbase32.Encode(randomBytes, bits)
}

func pubkeyFingerprintBase32(pubkey []byte) string {
	hash := sha256.Sum256(pubkey)
	fingerprint, _ := zbase32.Encode(hash[:], sha256.Size*8)
	return fingerprint
}

func pubkeyFingerprintMD5(pubkey []byte) string {
	hash := md5.Sum(pubkey)
	r := fmt.Sprintf("% x", hash)
	return strings.Replace(r, " ", ":", -1)
}
