// Copyright 2021-2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Webretc signal server client (for teonet network)
package teowebrtc_signal_client

import (
	"context"
	"encoding/json"
	"log"
	"net/url"

	"nhooyr.io/websocket"
)

// New signal server client
func New() *SignalClient {
	return new(SignalClient)
}

type SignalClient struct {
	conn   *websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

type Login struct {
	Signal string `json:"signal"`
	Login  string `json:"login"`
}

type Signal struct {
	Signal string      `json:"signal"`
	Peer   string      `json:"peer"`
	Data   interface{} `json:"data"`
}

// Connect to signal server and send login signal
func (cli *SignalClient) Connect(scheme, signalServerAddr, peerLogin string) (err error) {
	u := url.URL{Scheme: scheme, Host: signalServerAddr, Path: "/signal"}
	log.Printf("connecting to %s\n", u.String())
	ctx, cancel := context.WithCancel(context.Background())
	c, _, err := websocket.Dial(ctx, u.String(), nil)
	if err != nil {
		cancel()
		return
	}
	cli.conn = c
	cli.ctx = ctx
	cli.cancel = cancel

	// Send login signal
	var login = Login{"login", peerLogin}
	d, err := json.Marshal(login)
	if err != nil {
		log.Println("login marshal:", err)
		cli.Close()
		return
	}
	err = c.Write(ctx, websocket.MessageText, d)
	if err != nil {
		log.Println("write message error:", err)
		cli.Close()
		return
	}

	cli.waitAnswer()

	return
}

// Close connection to signal server
func (cli *SignalClient) Close() {
	log.Println("sinal client closed")
	cli.conn.Close(websocket.StatusNormalClosure, "done")
	cli.cancel()
}

// WaitSignal wait offer signal received
func (cli SignalClient) WaitSignal() (sig Signal, err error) {
	message, err := cli.waitAnswer()
	if err != nil {
		return
	}
	json.Unmarshal(message, &sig)
	return
}

// waitAnswer waite message received
func (cli SignalClient) waitAnswer() (message []byte, err error) {
	_, message, err = cli.conn.Read(cli.ctx)
	if err != nil {
		log.Println("read message error:", err)
	}
	return
}

// WriteOffer send offer signal
func (cli SignalClient) WriteOffer(peer string, offer interface{}) (answer []byte, err error) {
	log.Printf("send offer to %s", peer)
	err = cli.writeSignal("offer", peer, offer)
	if err != nil {
		return
	}
	answer, err = cli.waitAnswer()
	return
}

// WriteAnswer send answer signal
func (cli SignalClient) WriteAnswer(peer string, answer interface{}) (err error) {
	log.Printf("send answer to %s", peer)
	err = cli.writeSignal("answer", peer, answer)
	return
}

// WriteCandidate send candidate signal
func (cli SignalClient) WriteCandidate(peer string, candidate interface{}) (err error) {
	log.Printf("send ICECandidate to %s\n", peer)
	err = cli.writeSignal("candidate", peer, candidate)
	return
}

// writeSignal send signal
func (cli SignalClient) writeSignal(signal, peer string, data interface{}) (err error) {

	d, _ := json.Marshal(Signal{signal, peer, data})
	err = cli.conn.Write(cli.ctx, websocket.MessageText, d)
	if err != nil {
		log.Println("write message error:", err)
	}
	return
}
