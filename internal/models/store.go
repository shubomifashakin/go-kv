package models

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu sync.RWMutex
	Data map[string]string
	file *os.File
	Expiry map[string]int64
}

func (h *Store) Set(message string)string{
	h.mu.Lock()
	defer h.mu.Unlock()

	split:=strings.Split(message," ")
	key:=strings.TrimSpace(split[1])
	value:=strings.TrimSpace(split[2])
	h.Data[key]=value

	_,err:=h.file.WriteString(message)
	if err != nil {
		return "-ERR Internal server erorr \r\n"
	}

	return "+OK \r\n"
}

func (h *Store) Get(message string)string{
	h.mu.RLock()
	defer h.mu.RUnlock()

	split:=strings.Split(message," ")
	key:=strings.TrimSpace(split[1])

	value,exists:=h.Data[key]

	if !exists{
		return "-NOTFOUND key does not exist\r\n"
	}
	
	return fmt.Sprintf("$%d\r\n%s\r\n",len(value),value)
}

func (h *Store) Delete(message string)string{
	h.mu.Lock()
	defer h.mu.Unlock()

	split:=strings.Split(message," ")
	key:=strings.TrimSpace(split[1])

	if _, exists := h.Data[key]; !exists {
		return "-NOTFOUND key does not exist\r\n"
	}
	
	delete(h.Data,key)

	_,err:=h.file.WriteString(message)
	if err != nil {
		return "-ERR Internal server erorr \r\n"
	}

	return "+OK \r\n"
}

func (h *Store) Expire(message string) string{
	h.mu.Lock()
	defer h.mu.Unlock()

	split:= strings.Split(message," ")
	key:=strings.TrimSpace(split[1])
	seconds:= strings.TrimSpace(split[2])
	parsed,err:=strconv.Atoi(seconds)

	if err != nil {
		return "-INVALID ttl is not valid\r\n"
	}

	if _, exists := h.Data[key]; !exists {
		return "-NOTFOUND key does not exist\r\n"
	}

	duration:= time.Second * time.Duration(parsed)
	expiry:=time.Now().Add(duration).Unix()

	aofLine := fmt.Sprintf("EXP %s %d\r\n", key, expiry)
	_,err=h.file.WriteString(aofLine)

	if err != nil {
		return "-ERR Internal server erorr \r\n"
	}

	// add the key to the expiry list
	h.Expiry[key]= expiry

	return "+OK \r\n"
}

func (h *Store)StartSweep(ctx context.Context, wg *sync.WaitGroup){
	defer wg.Done()

	ticker:= time.NewTicker(time.Second*5)
	defer ticker.Stop()

	for {
		select {
			case <-ticker.C :
				h.mu.Lock()
				totalItems:=len(h.Expiry)
				

				if totalItems == 0 {
					h.mu.Unlock()
					continue
				}

				// for each item found, check if the current time is greater than its time
				for k,v:= range h.Expiry {
					if time.Now().Unix() < v {
						continue
					}

					delete(h.Expiry,k)
					delete(h.Data,k)
				}

				h.mu.Unlock()
			case <-ctx.Done():
				return
		}
	}
}


func NewStore(file *os.File) *Store{
	return &Store{
		file:file,
		Data: map[string]string{},
		Expiry: map[string]int64{},
	}
}