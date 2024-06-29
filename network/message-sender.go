package network

import (
	"bufio"
	"log"
)

// ReadFromNetWork Fonction permettant de lire en boucle sur un reader
func ReadFromNetWork(connReader *bufio.Reader, channel chan string) {

	for {
		msg, err := connReader.ReadString('\n')
		log.Print(msg)
		if err != nil {
			log.Panic(err)
		}
		channel <- msg[:len(msg)-1] // Ajoute le message au channel en retirant le dernier caractère (delimiter)
	}
}

func WriteFromNetWork(connWriter *bufio.Writer, channel chan string) {
	var message string
	for {
		message = <-channel
		_, err := connWriter.WriteString(message + "\n")
		if err != nil {
			log.Panic(err)
		}
		connWriter.Flush()
	}
}
