// Copyright 2009,2010, The 'gomongo' Authors.  All rights reserved.
// Use of this source code is governed by the 3-clause BSD License
// that can be found in the LICENSE file.

package mongo

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"
)


// Default Socket Port
//const _PORT = 27017


type Connection struct {
	Addr *net.TCPAddr
	conn *net.TCPConn
}

type Server struct {
	Host string
	Port int
}

const (
	MAX_AUTORECONNECTION_DELAY = 20 // seconds
)
var autoReconnectionDelay int64 = 1
var server Server
var disconnected bool = true

func Connect(host string, port int) (*Connection, os.Error) {
	return ConnectAt(host, port)
}

/* Creates a new connection to a single MongoDB instance at host:port. */
func ConnectAt(host string, port int) (*Connection, os.Error) {
	addr, err := net.ResolveTCPAddr(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	server.Host = host
	server.Port = port
	return ConnectByAddr(addr)
}

func autoReconnect(dbcon *Connection){
	fmt.Printf("Autoreconnect thread started\n")
	for {
		if disconnected == true {
			fmt.Printf("Database server disconnected. try to reconnect.\n")
			err := dbcon.ReconnectEx(dbcon)
			if err == nil{
				disconnected = false
				autoReconnectionDelay = 1
				fmt.Printf("Database connected.\n")
			}else{
				if autoReconnectionDelay < MAX_AUTORECONNECTION_DELAY{
					autoReconnectionDelay += 1
				}	
			}
		}
		if disconnected == true{
			fmt.Printf("Cannot reconnect. next reconnect within %d seconds\n", autoReconnectionDelay)
		}
		time.Sleep(autoReconnectionDelay * 1e9)
	}
}

func ConnectByAddr(addr *net.TCPAddr) (*Connection, os.Error) {
	// Connects from local host (nil)
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	
	dbcon := &Connection{addr, conn}
	
	disconnected = false
	go autoReconnect(dbcon)

	return dbcon, nil
}

func ConnectByAddrEx(addr *net.TCPAddr, dbcon *Connection) os.Error {
	// Connects from local host (nil)
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}
	
	dbcon.Addr = addr
	dbcon.conn = conn

	return nil
}

/* Reconnects using the same address `Addr`. */
func (self *Connection) Reconnect() (*Connection, os.Error) {
	connection, err := ConnectByAddr(self.Addr)
	if err != nil {
		return nil, err
	}

	return connection, nil
}

func (self *Connection) ReconnectEx(dbcon *Connection) os.Error{
	err := ConnectByAddrEx(self.Addr, dbcon)
	if err != nil {
		return err
	}
	return nil
}


/* Disconnects the conection from MongoDB. */
func (self *Connection) Disconnect() os.Error {
	if err := self.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (self *Connection) GetDB(name string) *Database {
	return &Database{self, name}
}

// === OP_REPLY

/* Gets the message of reply from database. */
func (self *Connection) readReply() (*opReply, os.Error) {
	size_bits, err := ioutil.ReadAll(io.LimitReader(self.conn, 4))
	if err != nil{return nil, err;}
	
	size := pack.Uint32(size_bits)
	rest, err := ioutil.ReadAll(io.LimitReader(self.conn, int64(size)-4))
	if err != nil{return nil, err;}
	
	reply := parseReply(rest)

	return reply, nil
}

