package esc

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var fileIn chan *File
var fileOut chan *File
var fileRm chan string
var newNode chan string

func startDirSync() {
	newNode = make(chan string, 4)
	fileIn = make(chan *File, 4)
	fileOut = make(chan *File, 4)
	fileRm = make(chan string, 4)

	OnEvent("*", "fileSync", onFileChanged)
	OnEvent("*", "fileRemoved", onFileRemoved)
	OnEvent("*", "connected", onNewNode)

	if config.Directory != "" {
		DirSync(config.Directory)
	}
}

func onNewNode(message *Message) {
	if message.Payload == Self() {
		return
	}
	log.Println("Sending files cause of new node", message.Payload)
	newNode <- message.Payload
}

func onFileRemoved(message *Message) {
	if message.From == Self() {
		return
	}
	fileRm <- message.Payload
}

func onFileChanged(message *Message) {
	parts := strings.SplitN(message.Payload, ",", 3)

	fileName := parts[0]

	time := time.Now()
	time.UnmarshalText([]byte(parts[1]))

	content, _ := base64.StdEncoding.DecodeString(parts[2])

	hasher := sha256.New()
	hasher.Write(content)
	hash := hex.EncodeToString(hasher.Sum(nil))

	file := File{
		Name:    fileName,
		Content: content,
		Time:    time,
		Hash:    hash,
	}

	fileIn <- &file
}

func ScanDir(registeredFiles map[string]*File, got map[string]bool, dirName string) {
	dir, _ := os.Open(dirName)
	files, _ := dir.Readdir(0)

	for _, fileInfo := range files {
		fileName := fmt.Sprintf("%s/%s", dirName, fileInfo.Name())

		if fileInfo.IsDir() {
			ScanDir(registeredFiles, got, fileName)
		} else if !fileInfo.IsDir() && !strings.HasPrefix(fileName, fmt.Sprintf("%s/.", dirName)) && !strings.HasSuffix(fileName, "~") {
			got[fileName] = true

			content, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Println(err)
				continue
			}

			hasher := sha256.New()
			hasher.Write(content)
			hash := hex.EncodeToString(hasher.Sum(nil))

			file := File{
				Name:    fileName,
				Content: content,
				Time:    fileInfo.ModTime(),
				Hash:    hash,
			}

			if _, ok := registeredFiles[fileName]; !ok {
				log.Println("Sending new file", fileName)
				go func() {
					fileOut <- &file
				}()
			} else if registeredFiles[fileName].Hash != hash {
				log.Println("Sending changed file", fileName)
				go func() {
					fileOut <- &file
				}()
			}

			registeredFiles[fileName] = &file
		}
	}

	for name, _ := range registeredFiles {
		if _, ok := got[name]; !ok {
			log.Println("Removed file", name)
			SendEvent("fileRemoved", name)
			fileRm <- name
		}
	}
}

func DirSync(dirName string) {
	registeredFiles := make(map[string]*File)

	ticker := time.NewTicker(6 * time.Second)

	go func() {
		_, errStat := os.Lstat(dirName)
		if errStat != nil {
			log.Println("Not syncing", dirName, errStat)
			return
		}

		for {
			select {
			case <-newNode:
				log.Println("new node, sending files")
				for _, file := range registeredFiles {
					go func() {
						fileOut <- file
					}()
				}
			case file := <-fileIn:
				log.Println("Got file", file.Name)
				if _, ok := registeredFiles[file.Name]; !ok {
					log.Println("New file", file.Name)
					ioutil.WriteFile(file.Name, file.Content, 0755)
					registeredFiles[file.Name] = file
				} else if registeredFiles[file.Name].Hash != file.Hash {
					log.Println("Changed file", file.Name)
					if registeredFiles[file.Name].Time.Before(file.Time) {
						log.Println("Their file is newer", file.Name)
						ioutil.WriteFile(file.Name, file.Content, 0755)
						registeredFiles[file.Name] = file
					} else {
						log.Println("Our", file.Name, "is older")
					}
				} else {
					log.Println("Same file", file.Name)
				}
			case file := <-fileRm:
				delete(registeredFiles, file)
				os.Remove(file)
			case file := <-fileOut:
				log.Println("Sending out file", file.Name)
				time, _ := file.Time.MarshalText()
				SendEventC("fileSync", fmt.Sprintf("%s,%s,%s", file.Name, time, base64.StdEncoding.EncodeToString(file.Content)), true)
			case <-ticker.C:
				got := make(map[string]bool)
				ScanDir(registeredFiles, got, dirName)
			}
		}
	}()
}
