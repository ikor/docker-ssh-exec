package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
)

func server(config Config) {

	// open receive port
	readSocket := openUDPSocket(`r`, net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: config.Port,
	})
	fmt.Printf("Listening on UDP port %d...\n", config.Port)
	defer readSocket.Close()

	// main loop
	for {
		data := make([]byte, UDP_MSG_SIZE)
		size, clientAddr, err := readSocket.ReadFromUDP(data)
		if err != nil {
			fmt.Println("Error reading from receive port: ", err)
		}
		clientMsg := data[0:size]
		if string(clientMsg) == KEY_REQUEST_TEXT {
			fmt.Printf("Received key request from %s, sending key.\n",
				clientAddr.IP)
			// reply to the client on the same port
			writeSocket := openUDPSocket(`w`, net.UDPAddr{
				IP:   clientAddr.IP,
				Port: clientAddr.Port + 1,
			})
			var keyData []byte
			keyData = []byte(os.Getenv(KEY_DATA_ENV_VAR))
			if len(keyData) == 0 {
				keyData, err = ioutil.ReadFile(config.KeyPath)
				if err != nil {
					fmt.Printf("ERROR reading keyfile %s: %s!\n", config.KeyPath, err)
				}
			}
			pemBlock, _ := pem.Decode(keyData)
			if pemBlock != nil {
				if x509.IsEncryptedPEMBlock(pemBlock) {
					fmt.Println("Decrypting private key with passphrase...")
					decoded, err := x509.DecryptPEMBlock(pemBlock, []byte(config.Pwd))
					if err == nil {
						header := `PRIVATE KEY` // default key type in header
						matcher := regexp.MustCompile("-----BEGIN (.*)-----")
						if matches := matcher.FindSubmatch(keyData); len(matches) > 1 {
							header = string(matches[1])
						}
						keyData = pem.EncodeToMemory(
							&pem.Block{Type: header, Bytes: decoded})
					} else {
						fmt.Printf("Error decrypting PEM-encoded secret: %s\n", err)
					}
				}
			}
			_, err = writeSocket.Write(keyData)
			if err != nil {
				fmt.Printf("ERROR writing data to socket:%s!\n", err)
			}
			writeSocket.Close()
		}
	}
}
