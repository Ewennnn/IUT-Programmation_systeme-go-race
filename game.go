/*
//  Data structure for representing a game. Implements the ebiten.Game
//  interface (Update in game-update.go, Draw in game-draw.go, Layout
//  in game-layout.go). Provided with a few utilitary functions:
//    - initGame
*/

package main

import (
	"bufio"
	"bytes"
	"course/assets"
	"fmt"
	"gitlab.univ-nantes.fr/E22B127S/projet-net-code/network"
	"image"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	state         int           // Current state of the game
	runnerImage   *ebiten.Image // Image with all the sprites of the runners
	runners       [4]Runner     // The four runners used in the game
	f             Field         // The running field
	launchStep    int           // Current step in StateLaunchRun state
	resultStep    int           // Current step in StateResult state
	getTPS        bool          // Help for debug
	readChan      chan string
	writeChan     chan string
	clientsCount  int
	readyNextStep bool
	responseSend  bool
	clientId      int
	finalTimes    [4]time.Duration
}

// These constants define the five possible states of the game
const (
	StateWelcomeScreen int = iota // Title screen
	StateChooseRunner             // Player selection screen
	StateLaunchRun                // Countdown before a run
	StateRun                      // Run
	StateResult                   // Results announcement
)

// InitGame builds a new game ready for being run by ebiten
func InitGame(ip, port string) (g Game) {

	// Open the png image for the runners sprites
	img, _, err := image.Decode(bytes.NewReader(assets.RunnerImage))
	if err != nil {
		log.Fatal(err)
	}
	g.runnerImage = ebiten.NewImageFromImage(img)

	// Define game parameters
	start := 50.0
	finish := float64(screenWidth - 50)
	frameInterval := 20

	// Create the runners
	for i := range g.runners {
		//interval := 20
		//if i == 0 {
		//	interval = frameInterval
		//}
		g.runners[i] = Runner{
			xpos: start, ypos: 50 + float64(i*20),
			maxFrameInterval: frameInterval,
			colorScheme:      i,
		}
	}

	// Create the field
	g.f = Field{
		xstart:   start,
		xarrival: finish,
		chrono:   time.Now(),
	}

	// Open connection
	log.Println(ip + ":" + port)
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		log.Fatal(err)
	}

	// Initialisation du channel de communication
	g.readChan = make(chan string, 1)
	g.writeChan = make(chan string, 1)

	// Goroutine écoutant permettant le lire en double sur un reader initialisé avec la connection
	go network.ReadFromNetWork(bufio.NewReader(conn), g.readChan)
	go network.WriteFromNetWork(bufio.NewWriter(conn), g.writeChan)

	var message = <-g.readChan
	if message[:1] == network.CLIENT_NUMBER {
		var idFromServ, _ = strconv.Atoi(message[1:])
		g.clientId = idFromServ
		ebiten.SetWindowTitle("BUT2 année 2022-2023, R3.05 Programmation système, clientID: " + fmt.Sprint(g.clientId))
	}

	g.writeChan <- network.CLIENT_CONNECTED

	return g
}
