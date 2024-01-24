// Copyright 2023 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"

	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/vitess/go/mysql"

	_ "github.com/dolthub/doltgresql/server/functions"
)

var (
	connectionIDCounter uint32
	processID           = int32(os.Getpid())
	certificate         tls.Certificate //TODO: move this into the mysql.ListenerConfig
)

// Listener listens for connections to process PostgreSQL requests into Dolt requests.
type Listener struct {
	listener net.Listener
	cfg      mysql.ListenerConfig
}

var _ server.ProtocolListener = (*Listener)(nil)

// NewListener creates a new Listener.
func NewListener(listenerCfg mysql.ListenerConfig) (server.ProtocolListener, error) {
	return &Listener{
		listener: listenerCfg.Listener,
		cfg:      listenerCfg,
	}, nil
}

// Accept handles incoming connections.
func (l *Listener) Accept() {
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			if err.Error() == "use of closed network connection" {
				break
			}
			fmt.Printf("Unable to accept connection:\n%v\n", err)
			continue
		}

		connectionHandler := NewConnectionHandler(conn, l.cfg.Handler)
		go connectionHandler.HandleConnection()
	}
}

// Close stops the handling of incoming connections.
func (l *Listener) Close() {
	_ = l.listener.Close()
}

// Addr returns the address that the listener is listening on.
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}
