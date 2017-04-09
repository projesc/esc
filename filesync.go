package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var registeredFiles map[string]*File

func startDirSync() {
	OnEvent("*", "fileSync", onFileChanged)
	OnEvent("*", "fileRemoved", onFileRemoved)
	if config.Scripts != "" {
		dirSync(config.Scripts)
	}
}

type File struct {
	Name string
	Hash string
}

func sendFile(file string) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println(err)
	} else {
		SendEvent("fileSync", fmt.Sprintf("%s,%s", file, string(content)))
	}
}

func removeFile(file string) {
	SendEvent("fileRemoved", file)
}

func onFileRemoved(message *Message) {
	if message.From == Self() {
		return
	}

	delete(registeredFiles, message.Payload)
	os.Remove(message.Payload)
}

func onFileChanged(message *Message) {
	if message.From == Self() {
		return
	}

	parts := strings.SplitN(message.Payload, ",", 2)
	fileName := parts[0]
	content := parts[1]
	hasher := sha256.New()
	hasher.Write([]byte(content))
	hash := hex.EncodeToString(hasher.Sum(nil))

	if _, ok := registeredFiles[fileName]; ok {
		if registeredFiles[fileName].Hash == hash {
			return
		} else {
			registeredFiles[fileName].Hash = hash
			ioutil.WriteFile(fileName, []byte(content), 0755)
		}
	} else {
		file := File{Name: fileName, Hash: hash}
		registeredFiles[fileName] = &file
		ioutil.WriteFile(fileName, []byte(content), 0755)
	}
}

func dirSync(dirName string) {
	ticker := time.NewTicker(5 * time.Second)
	registeredFiles = make(map[string]*File)

	go func() {
		for {
			<-ticker.C
			_, errStat := os.Lstat(dirName)
			if errStat != nil {
				log.Println(errStat)
				continue
			}
			dir, _ := os.Open(dirName)

			files, err := dir.Readdir(0)

			if err != nil {
				log.Println(err)
				continue
			}

			got := make(map[string]bool)

			for _, fileInfo := range files {
				fileName := fmt.Sprintf("%s/%s", dirName, fileInfo.Name())
				if strings.HasPrefix(fileName, fmt.Sprintf("%s/.", dirName)) {
					continue
				}
				got[fileName] = true

				hasher := sha256.New()
				f, _ := os.Open(fileName)
				io.Copy(hasher, f)
				f.Close()

				hash := hex.EncodeToString(hasher.Sum(nil))
				if _, ok := registeredFiles[fileName]; ok {
					if registeredFiles[fileName].Hash == hash {
						// not changed
					} else {
						log.Println("Updating file", fileName)
						registeredFiles[fileName].Hash = hash
						sendFile(fileName)
					}
				} else {
					log.Println("Sending file", fileName)
					file := File{Name: fileName, Hash: hash}
					registeredFiles[fileName] = &file
					sendFile(fileName)
				}
			}

			var toRemove []string

			for name, _ := range registeredFiles {
				if _, ok := got[name]; ok {
				} else {
					log.Println("Removing file", name)
					toRemove = append(toRemove, name)
				}
			}

			for _, name := range toRemove {
				removeFile(name)
				delete(registeredFiles, name)
			}
		}
	}()
}
