package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var fileIn chan *File
var fileRm chan string
var newNode chan string

type File struct {
	Name    string
	Hash    string
	Content string
}

func startDirSync() {
	newNode = make(chan string)
	fileIn = make(chan *File)
	fileRm = make(chan string)

	OnEvent("*", "fileSync", onFileChanged)
	OnEvent("*", "fileRemoved", onFileRemoved)
	OnEvent("*", "connected", onNewNode)

	if config.Scripts != "" {
		DirSync(config.Scripts)
	}
}

func onNewNode(message *Message) {
	if string(message.Payload) == Self() {
		return
	}
	log.Println("Sending files cause of new node", string(message.Payload))
	newNode <- string(message.Payload)
}

func onFileRemoved(message *Message) {
	if message.From == Self() {
		return
	}
	fileRm <- string(message.Payload)
}

func onFileChanged(message *Message) {
	if message.From == Self() {
		return
	}

	parts := strings.SplitN(string(message.Payload), ",", 2)
	fileName := parts[0]
	content := parts[1]
	hasher := sha256.New()
	hasher.Write([]byte(content))
	hash := hex.EncodeToString(hasher.Sum(nil))

	file := File{
		Name:    fileName,
		Content: content,
		Hash:    hash,
	}
	fileIn <- &file
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
				log.Println("New node, sending files")
				for _, file := range registeredFiles {
					SendEventC("fileSync", []byte(fmt.Sprintf("%s,%s", file.Name, file.Content)), true)
				}
			case file := <-fileIn:
				log.Println("Got new file", file.Name)
				if _, ok := registeredFiles[file.Name]; !ok {
					ioutil.WriteFile(file.Name, []byte(file.Content), 0755)
				} else if registeredFiles[file.Name].Hash != file.Hash {
					ioutil.WriteFile(file.Name, []byte(file.Content), 0755)
				}
				registeredFiles[file.Name] = file
			case file := <-fileRm:
				delete(registeredFiles, file)
				os.Remove(file)
			case <-ticker.C:
				dir, _ := os.Open(dirName)
				files, _ := dir.Readdir(0)
				got := make(map[string]bool)

				for _, fileInfo := range files {
					fileName := fmt.Sprintf("%s/%s", dirName, fileInfo.Name())

					if !strings.HasPrefix(fileName, fmt.Sprintf("%s/.", dirName)) && !strings.HasSuffix(fileName, "~") {
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
							Content: string(content),
							Hash:    hash,
						}

						if _, ok := registeredFiles[fileName]; !ok {
							log.Println("Sending new file", fileName)
							SendEventC("fileSync", []byte(fmt.Sprintf("%s,%s", file.Name, file.Content)), true)
						} else if registeredFiles[fileName].Hash != hash {
							log.Println("Sending changed file", fileName)
							SendEventC("fileSync", []byte(fmt.Sprintf("%s,%s", file.Name, file.Content)), true)
						}
						registeredFiles[fileName] = &file
					}
				}

				for name, _ := range registeredFiles {
					if _, ok := got[name]; !ok {
						log.Println("Removed file", name)
						SendEvent("fileRemoved", []byte(name))
						fileRm <- name
					}
				}
			}
		}
	}()
}
