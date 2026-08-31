package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	flag.NewFlagSet("get",flag.ExitOnError)
	flag.NewFlagSet("delete",flag.ExitOnError)
	flag.NewFlagSet("ping",flag.ExitOnError)
	setFlagset:=flag.NewFlagSet("set",flag.ExitOnError)
	expireFlagset:=flag.NewFlagSet("expire",flag.ExitOnError)
	
	setKey:=setFlagset.String("k","","The key to use")
	setVal:=setFlagset.String("v","","The value to associate with the key")

	expKey:=expireFlagset.String("k","","The key to expire")
	expSeconds:=expireFlagset.Int("t",0,"The amount of time in seconds that the key has to live")

	conn, err := net.Dial("tcp", "localhost:3001")
	if err != nil {
		log.Fatalln("Failed to connect to server",err)
	}
	
	defer conn.Close()

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "ping":
		_,err:=fmt.Fprintf(conn,"PING\r\n")
		if err != nil {
			log.Fatalln("failed to connect",err)
		}

		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to read response",err)
		}

		responseFormatter(response)

	case "get":
		key := os.Args[2]

		if (key == ""){
			log.Fatalln("key is required")
			return
		}
		
		_,err:=fmt.Fprintf(conn, "GET %s\r\n", key)
		if err != nil {
			log.Fatalln("Failed to get key",err)
		}
	
		reader := bufio.NewReader(conn)
		first, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to read response",err)
		}
	
		if first[0] == '-' {
			fmt.Fprintln(os.Stderr, strings.TrimSpace(first[1:]))
			return
		}

		value, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to get value",err)
		}
		fmt.Println(strings.TrimSpace(value))
	

	case "delete":
		key:= os.Args[2]

		if (key == ""){
			log.Fatalln("key is required")
			return
		}

		_,err:=fmt.Fprintf(conn, "DEL %s\r\n",key)
		if err != nil {
			log.Fatalln("Failed to delete key",err)
		}

		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to write response",err)
		}
		responseFormatter(response)

	case "set":
		setFlagset.Parse(os.Args[2:])
		key:=*setKey
		value:=*setVal

		if(key == "" || value== ""){
			log.Fatalln("key and value are required")
			return
		}

		_,err:=fmt.Fprintf(conn, "SET %s %s\r\n",key,value)
		if err != nil {
			log.Fatalln("Failed to set key",err)
		}

		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to write response",err)
		}
		responseFormatter(response)

	case "expire":
		expireFlagset.Parse(os.Args[2:])

		key:=*expKey
		seconds:=*expSeconds

		if(key == "" || seconds== 0){
			log.Fatalln("key and seconds are required")
			return
		}

		expireAt := time.Now().Add(time.Duration(seconds) * time.Second).Unix()
		_,err:=fmt.Fprintf(conn, "EXP %s %d\r\n", key, expireAt)
		if err != nil {
			log.Fatalln("Failed to expire key",err)
		}

		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalln("Failed to write response",err)
		}

		responseFormatter(response)
	default:
		printUsage()
	}
}

func responseFormatter(response string) {
	switch {
	case response == "+OK \r\n":
		fmt.Println("OK")
	case response == "+PONG \r\n":
		fmt.Println("PONG")
	case len(response) > 0 && response[0] == '$':
		fmt.Print(response[1:])
	case len(response) > 0 && response[0] == '-':
		fmt.Fprintln(os.Stderr, response[1:])
	default:
		fmt.Print(response)
	}
}

func printUsage() {
	fmt.Println("Usage: kv <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  ping                      Check if the server is alive")
	fmt.Println("  get <key>                 Get the value of a key")
	fmt.Println("  set -k <key> -v <value>   Set a key-value pair")
	fmt.Println("  delete <key>              Delete a key")
	fmt.Println("  expire -k <key> -t <sec>  Set a TTL on a key in seconds")
}
