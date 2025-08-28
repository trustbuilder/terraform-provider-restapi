package apiclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	apiClientServer             *http.Server
	apiClientTLSServer          *http.Server
	rootCA                      *x509.Certificate
	rootCAKey                   *ecdsa.PrivateKey
	serverCertPEM, serverKeyPEM []byte
	rootCAFilePath              = "rootCA.pem"
)

func TestAPIClient(t *testing.T) {
	debug := false

	if debug {
		log.Println("client_test.go: Starting HTTP server")
	}
	setupAPIClientServer()

	/* Notice the intentional trailing / */
	opt := &ApiClientOpt{
		Uri:                 "http://127.0.0.1:8083/",
		Insecure:            false,
		Username:            "",
		Password:            "",
		Headers:             make(map[string]string),
		Timeout:             2,
		IdAttribute:         "id",
		CopyKeys:            make([]string, 0),
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
		RateLimit:           1,
		Debug:               debug,
	}
	client, _ := NewAPIClient(opt)

	var res string
	var err error

	if debug {
		log.Printf("api_client_test.go: Testing standard OK request\n")
	}
	res, err = client.SendRequest("GET", "/ok", "")
	if err != nil {
		t.Fatalf("client_test.go: %s", err)
	}
	if res != "It works!" {
		t.Fatalf("client_test.go: Got back '%s' but expected 'It works!'\n", res)
	}

	if debug {
		log.Printf("api_client_test.go: Testing redirect request\n")
	}
	res, err = client.SendRequest("GET", "/redirect", "")
	if err != nil {
		t.Fatalf("client_test.go: %s", err)
	}
	if res != "It works!" {
		t.Fatalf("client_test.go: Got back '%s' but expected 'It works!'\n", res)
	}

	/* Verify timeout works */
	if debug {
		log.Printf("api_client_test.go: Testing timeout aborts requests\n")
	}
	_, err = client.SendRequest("GET", "/slow", "")
	if err == nil {
		t.Fatalf("client_test.go: Timeout did not trigger on slow request")
	}

	if debug {
		log.Printf("api_client_test.go: Testing rate limited OK request\n")
	}
	startTime := time.Now().Unix()

	for i := 0; i < 4; i++ {
		_, err = client.SendRequest("GET", "/ok", "")
		if err != nil {
			t.Fatalf("client_test.go: Timeout did not trigger on ok request")
		}
	}

	duration := time.Now().Unix() - startTime
	if duration < 3 {
		t.Fatalf("client_test.go: requests not delayed\n")
	}

	if debug {
		log.Println("client_test.go: Stopping HTTP server")
	}
	shutdownAPIClientServer()
	if debug {
		log.Println("client_test.go: Done")
	}

	// Setup and test HTTPS client with root CA
	setupAPIClientTLSServer()
	defer shutdownAPIClientTLSServer()
	defer os.Remove(rootCAFilePath)

	httpsOpt := &ApiClientOpt{
		Uri:                 "https://127.0.0.1:8443/",
		Insecure:            false,
		Username:            "",
		Password:            "",
		Headers:             make(map[string]string),
		Timeout:             2,
		IdAttribute:         "id",
		CopyKeys:            make([]string, 0),
		WriteReturnsObject:  false,
		CreateReturnsObject: false,
		RateLimit:           1,
		RootCaFile:          rootCAFilePath,
		Debug:               debug,
	}
	httpsClient, httpsClientErr := NewAPIClient(httpsOpt)

	if httpsClientErr != nil {
		t.Fatalf("client_test.go: %s", httpsClientErr)
	}
	if debug {
		log.Printf("api_client_test.go: Testing HTTPS standard OK request\n")
	}
	res, err = httpsClient.SendRequest("GET", "/ok", "")
	if err != nil {
		t.Fatalf("client_test.go: %s", err)
	}
	if res != "It works!" {
		t.Fatalf("client_test.go: Got back '%s' but expected 'It works!'\n", res)
	}
}

func setupAPIClientServer() {
	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("It works!")); err != nil {
			log.Fatalf("client_test.go: Error on sending ok response: %s\n", err)
		}
	})
	serverMux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(9999 * time.Second)
		if _, err := w.Write([]byte("This will never return!!!!!")); err != nil {
			log.Fatalf("client_test.go: Error on sending slow response: %s\n", err)
		}
	})
	serverMux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ok", http.StatusPermanentRedirect)
	})

	apiClientServer = &http.Server{
		Addr:    "127.0.0.1:8083",
		Handler: serverMux,
	}

	go func() {
		if err := apiClientServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("setupAPIClientServer(): %v", err)
		}
	}()
	/* let the server start */
	time.Sleep(1 * time.Second)
}

func shutdownAPIClientServer() {
	apiClientServer.Close()
}

func setupAPIClientTLSServer() {
	generateCertificates()

	cert, _ := tls.X509KeyPair(serverCertPEM, serverKeyPEM)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("It works!"))
	})

	apiClientTLSServer = &http.Server{
		Addr:      "127.0.0.1:8443",
		Handler:   serverMux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	go apiClientTLSServer.ListenAndServeTLS("", "")
	/* let the server start */
	time.Sleep(1 * time.Second)
}

func shutdownAPIClientTLSServer() {
	apiClientTLSServer.Close()
}

func generateCertificates() {
	// Create a CA certificate and key
	rootCAKey, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	rootCA = &x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject: pkix.Name{
			Organization: []string{"Test Root CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	rootCABytes, _ := x509.CreateCertificate(rand.Reader, rootCA, rootCA, &rootCAKey.PublicKey, rootCAKey)

	// Create a server certificate and key
	serverKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	serverCert := &x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject: pkix.Name{
			Organization: []string{"Test Server"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	// Add IP SANs to the server certificate
	serverCert.IPAddresses = append(serverCert.IPAddresses, net.ParseIP("127.0.0.1"))

	serverCertBytes, _ := x509.CreateCertificate(rand.Reader, serverCert, rootCA, &serverKey.PublicKey, rootCAKey)

	// PEM encode the certificates and keys
	serverCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertBytes})

	// Marshal the server private key
	serverKeyBytes, _ := x509.MarshalECPrivateKey(serverKey)
	serverKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverKeyBytes})

	rootCAPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCABytes})
	_ = os.WriteFile(rootCAFilePath, rootCAPEM, 0644)
}
