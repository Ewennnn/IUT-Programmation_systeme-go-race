package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"network"
	"strconv"
	"sync"
)

// Server : Structure de données avec tout ce qu'il nous faut !
type Server struct {
	clients  []net.Conn // Tableau de connexions qui représentent les clients
	listener net.Listener

	writeChans []chan string // Tunnel d'écriture
	readChans  []chan string // Tunnel de lecture

	countRunners    int        // Compteur utilisé plusieurs fois dans le programme
	mutexCountRC    sync.Mutex // Mutex permettant de bloquer certaines parties du code
	finishTimes     [4]string  // Tableau des temps
	runnerPositions [4]int     // Tableau des runners sélectionnés
}

// newServer : Creating a new server with clients, listener, writeChans, readChans
func newServer() Server {
	listener, _ := net.Listen("tcp", ":8080")
	return Server{
		[]net.Conn{},
		listener,
		[]chan string{},
		[]chan string{},
		0,
		sync.Mutex{},
		[4]string{},
		[4]int{0, 1, 2, 3},
	}
}

// acceptClients : Accepte 4 connexions de clients
func (s *Server) acceptClients() {
	for i := 0; i < 4; i++ { // Pour 4 connections
		conn, err := s.listener.Accept() // Accepte la connection du client
		if err != nil {                  // La méthode Accept est bloquante, elle ne passe à l'étape suivante (ligne suivante) que lorsqu'une connection vient d'être acceptée
			log.Fatal(err)
		}
		s.clients = append(s.clients, conn) // Ajout de la connection de ce nouveau client à la liste des clients

		// Créer les canaux de communication avec ce client
		// Enregistre ces canaux dans un tableau répertoriant tous les canaux
		s.writeChans = append(s.writeChans, make(chan string, 10))
		s.readChans = append(s.readChans, make(chan string, 10))

		// Initialiser les goroutines de communication de ce client avec la connection et le channel initialisé précédement
		go network.ReadFromNetWork(bufio.NewReader(conn), s.readChans[i])
		go network.WriteFromNetWork(bufio.NewWriter(conn), s.writeChans[i])

		s.writeChans[i] <- network.CLIENT_NUMBER + fmt.Sprint(i) // Envoyer l'ID au client qui vient de se connecter
		s.sendToAll(network.CLIENTS_IN_QUEUE + fmt.Sprint(i+1))  // Envoie à tout le monde le nombre de clients actuellement connectés

		log.Println("Client ", i, "connected")
		log.Println("in queue: ", fmt.Sprint(i+1))
	}
}

// sendToAll : Envoyer un message à tous les clients
func (s *Server) sendToAll(message string) {
	for _, chann := range s.writeChans { // Pour tous les channels d'écriture ouverts
		chann <- message // On envoie le message dans ce canal
	}
}

// comClient : Communication avec un client
func (s *Server) comClient(clientID int) {
	var stringClientID = fmt.Sprint(clientID)
	var message string
	for {
		message = <-s.readChans[clientID] // On lit le message du channel du client
		// On rappelle que cette fonction est lancée comme go routine et que le serveur lance cette fonction pour chaque client
		switch message[:1] {
		// Cas lorsqu'un client vient de se connecter
		case network.CLIENT_CONNECTED:
			log.Println("client ", clientID, "connecté")

		case network.RUNNER_CHOICE_POSITION:
			if message[1:2] == stringClientID { // Si l'identifiant du client reçu est celui du client géré par la goroutine
				var direction = message[2:3]        // On récupère la direction vers laquelle le client effectuait sa sélection
				pos, _ := strconv.Atoi(message[3:]) // On récupère la position du curseur de sélection
				s.mutexCountRC.Lock()
				for contains(&s.runnerPositions, pos) { // Tant que la position du curseur est sur un runner déjà sélectionné
					log.Println("Position " + fmt.Sprint(pos) + " dejà pris par le client " + fmt.Sprint(findIndex(&s.runnerPositions, pos)))
					if direction == "L" { // Si la sélection s'effectue par la gauche
						pos = (pos + 7) % 8 // On calcule la position du curseur vers cette direction
					}
					if direction == "R" { // Si la sélection s'effectue par la droite
						pos = (pos + 1) % 8 // On calcule la position du curseur vers cette direction
					}
				}
				s.runnerPositions[clientID] = pos // On enregistre le nouveau runner sélectionné par ce client
				s.mutexCountRC.Unlock()
				s.sendToAll(network.RUNNER_CHOICE_POSITION + stringClientID + " " + fmt.Sprint(pos)) // On envoie à tous les clients le runner sélectionné pour clientID
				log.Println(s.runnerPositions)
			}

		// Cas lorsqu'un client vient de sélectionner un joueur
		case network.CLIENT_CHOOSE_RUNNER:
			var selected = message[1:]
			s.mutexCountRC.Lock() // On bloque la partie du code suivante
			if selected == "0" {  // Le client a déselectionné son runner
				s.countRunners--
				log.Println("client", clientID, "has deselected his runner")
			}
			if selected == "1" { // Le client a sélectionné son runner
				s.countRunners++ // On compte le nombre de joueurs ayant choisi leur personnage
				log.Println("client", clientID, "chose his runner")
			}
			s.sendToAll(network.CLIENTS_IN_QUEUE + fmt.Sprint(s.countRunners)) // On envoie à tous les clients le nombre de joueurs ayant choisi leur personnage
			if s.countRunners == 4 {                                           // Si les 4 joueurs ont choisi leur personnage
				s.sendToAll(network.ALL_RUNNER_CHOOSEN) // On envoie à tous les clients que tout le monde a choisi son personnage
				s.sendToAll(network.START_RACE)         // On envoie à tous les clients que la course commence

				s.countRunners = 0 // On réinitialise notre compteur
			}
			s.mutexCountRC.Unlock() // On débloque cette partie du code

		// Cas lorsqu'un joueur a terminé sa course
		case network.FINISH_RACE:
			s.finishTimes[clientID] = message[1:] // On récupère le temps d'un joueur en récupérant le message reçu sans le premier caractère

			var playersFinishedRace = 0
			for _, time := range s.finishTimes { // On vérifie que tous les clients aient renvoyé leur temps
				if time != "" { // Si le temps d'un joueur n'est pas vide
					playersFinishedRace++ // On le compte comme ayant renvoyé son temps
				}
			}
			if playersFinishedRace == 4 { // Si tous les joueurs ont renvoyé leur temps
				var text = network.FINISH_RACE       // On initialise le texte
				for _, time := range s.finishTimes { // Pour les temps de chaque joueur
					text += time + " " // On ajoute ce temps à une chaine de caractères
				}
				s.sendToAll(text[:len(text)-1]) // On envoie cette chaine de caractères contenant tous les temps des joueurs et on l'envoie à tous les clients.
			}

		// Cas lorsqu'un joueur souhaite relancer une partie
		case network.CLIENT_WISH_RESTART:
			s.mutexCountRC.Lock()    // On bloque cette partie de code
			s.countRunners++         // On compte les joueurs qui souhaitent relancer une partie
			if s.countRunners == 4 { // Si 4 joueurs souhaitent relancer une partie
				s.sendToAll(network.START_RACE) // On envoie à tous les clients qu'une nouvelle partie recommence.
				s.sendToAll(network.START_RACE) // Deux fois, car avec un seul envoie certains clients ne relancent pas la partie (bug).
				// Une fois la course relancée
				s.countRunners = 0          // On réinitialise le compteur
				s.finishTimes = [4]string{} // On réinitialise le tableau des temps
				s.mutexCountRC.Unlock()     // On débloque la partie du code
				break                       // On sort de la boucle for
			}
			// Si tous les joueurs n'ont pas souhaité relancer une partie
			s.sendToAll(network.CLIENTS_IN_QUEUE + fmt.Sprint(s.countRunners)) // On envoie à tous les clients le nombre de clients souhaitant relancer une partie
			s.mutexCountRC.Unlock()                                            // On débloque le bloc de code
		case network.NEED_CLIENT_COUNT:
			s.writeChans[clientID] <- network.CLIENTS_IN_QUEUE + fmt.Sprint(s.countRunners)

		case network.RUNNER_POSITION:
			if message[1:2] == stringClientID {
				log.Println("Position et vitesse du runner ", clientID, " : ", message[2:])
				s.sendToAll(network.RUNNER_POSITION + stringClientID + fmt.Sprint(message[2:]))
			}
		}

	}
}

func main() {
	var server = newServer() // Création du serveur
	log.Println("Listening for connections")
	server.acceptClients() // Attend que les 4 clients soient connectés
	// Le programme ne passe pas à l'étape suivante (ligne suivante) tant que 4 clients ne se sont pas connectés
	server.sendToAll(network.ALL_CONNECTED) // Envoie à tous les clients l'information que tous les clients sont connectés
	log.Println("All clients connected")
	defer server.listener.Close()         // Fermer la connection à la fin du programme
	for i, conn := range server.clients { // Pour tous les clients connectés
		defer conn.Close()
		go server.comClient(i) // On initialise une connection avec ce client
	}

	for {
	} // Pour empêcher le serveur de s'éteindre
}

func contains(s *[4]int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func findIndex(s *[4]int, e int) int {
	for i, a := range s {
		if a == e {
			return i
		}
	}
	return -1
}
