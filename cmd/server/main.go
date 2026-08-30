package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/shubomifashakin/go-kv/internal/models"
	"github.com/shubomifashakin/go-kv/pkg/utils"
	"go.uber.org/zap"
)

func main() {
	logger,err:=zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	appEnv:= os.Getenv("APP_ENV")

	if err:=godotenv.Load(); err != nil && appEnv != "production" {
		logger.Error("Failed to load environment variables",zap.Error(err))
	}

	// validate the environment variables
	cfg,err:=utils.ValidateServerConfig()

	if err != nil {
		logger.Fatal("Server not configured properly",zap.Error(err))
	}

	logger.WithOptions(zap.Fields(zap.String("environment",appEnv)))

	shutdownCtx, shutdown := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer shutdown()

	// rebuild the store
	store,err:=rebuildStore()

	if err != nil {
		logger.Fatal("Failed to rebuild store",zap.Error(err))
	}

	logger.Info(fmt.Sprintf("Starting server on port %s",cfg.Port))

	// start the tcp server
	listener,err:=net.Listen("tcp",":"+cfg.Port)

	if err != nil {
		logger.Fatal("Failed to start tcp server",zap.Error(err))
	}

	go handleSignal(shutdownCtx,logger,listener)

	var wg sync.WaitGroup

	// start the worker that cleans up expired messages
	wg.Add(1)
	go store.StartSweep(shutdownCtx,&wg)

	// loop through each connection received
	for {
		conn, err := listener.Accept()

		// if the shutdown signal has already been receiveed stop accepting connections
		if shutdownCtx.Err() != nil {
			wg.Wait()
			return
		}

		if err != nil {
			logger.Error("Failed to accept:", zap.Error(err))
			continue
		}

		wg.Add(1)
			
		go handleConnection(&wg,conn,logger,store)
	}
}


func handleConnection(wg *sync.WaitGroup,conn net.Conn, logger *zap.Logger, store *models.Store){
	defer wg.Done()

	timeReceived:= time.Now()
	
	// data must be read within 30 seconds of being received
	readTimeout:= time.Second*30
	conn.SetReadDeadline(timeReceived.Add(readTimeout))

	// a response must be written within 1 minute of the data being received
	writeTimeout:= time.Minute
	conn.SetWriteDeadline(timeReceived.Add(writeTimeout))

	// read the data sent from the client
	reader := bufio.NewReader(conn)
	for {
		// this errors when the connection is closed
		message, err := reader.ReadString('\n')
		if err != nil {
			var netErr net.Error

			if errors.As(err, &netErr) && netErr.Timeout() {
				logger.Debug("Client idle timeout, closing connection")
			} else {
				logger.Error("Failed to read from connection", zap.Error(err))
			}
			conn.Close()
			return
		}
		
		// after each successful read the client has 30 seconds to make their next request
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		
		// get the command
		command := strings.TrimSpace(strings.Split(message, " ")[0])

		logger.Debug("Received message", zap.String("message", message), zap.String("command", command))

		// handle the command
		switch command {
			case "SET":
				resp:=store.Set(message)
				conn.Write([]byte(resp))

			case "GET":
				resp:=store.Get(message)
				conn.Write([]byte(resp))

			case "DEL":
				resp:=store.Delete(message)
				conn.Write([]byte(resp))

			case "EXP":
				resp:=store.Expire(message)
				conn.Write([]byte(resp))

			case "PING":
				conn.Write([]byte("+PONG \r\n"))

			default:
				response:="-INVALID unknown command\r\n"
				logger.Error("Invalid command received",zap.String("command",command))
				conn.Write([]byte(response))
		}

		conn.SetWriteDeadline(time.Now().Add(readTimeout))
	}
}

// rebuilds the store from the append only file
func rebuildStore()(*models.Store,error){	
	homeDir,err:=os.UserHomeDir()
	if err != nil {
		return nil,err
	}

	// create the aof directory
	aofDir:= filepath.Join(homeDir,".go-kv")
	err=os.MkdirAll(aofDir,0755)

	if err != nil {
		return nil,err
	}

	// open the file for reading
	aofPath:=filepath.Join(aofDir,"data.aof")
	file,err:= os.OpenFile(aofPath,os.O_CREATE|os.O_RDWR|os.O_APPEND,0644)
	
	// if an error occurred
	if err != nil && !os.IsNotExist(err) {
		return nil,err
	}
	
	// if the file exists, rebuild the store
	records:=models.NewStore(file)
	if err == nil {
		reader:=bufio.NewReader(file)
			
		// loop through the entries
		for{
			line,err:=reader.ReadString('\n')

			// if an error ocurred while reading the file
			if err != nil && err !=io.EOF {
				return nil,err
			}
			
			// if it has finished reading the file
			if err != nil {
				break
			}

			// get the command used
			entries:= strings.Split(line," ")
			command:= entries[0]
			key:=entries[1]
			var value string

			if command == "SET" {
				value= entries[2]
				records.Data[key]=strings.TrimSpace(value)
			}

			if command == "DEL" {
				delete(records.Data,key)
			}

			if command == "EXP" {
				expSeconds:=strings.TrimSpace(entries[2])
				expAt, err := strconv.Atoi(strings.TrimSpace(expSeconds))
				if err != nil {
					return nil, err
				}

				// if the entry has already expired, just delete it
				if time.Now().Unix() > int64(expAt) {
					delete(records.Data, key)
				}else{
				// add it to the queue
					records.Expiry[key]=int64(expAt)
				}
			}
		}
	}

	return records,nil
}

func handleSignal(ctx context.Context,logger *zap.Logger,listener net.Listener){
	<-ctx.Done()

	logger.Info("Shutting down server")
	
	listener.Close()
}