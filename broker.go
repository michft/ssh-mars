package main

type Broker struct {
	Notifier       chan string
	newClients     chan chan string
	closingClients chan chan string
	clients        map[chan string]struct{}
}

func NewBroker() (broker *Broker) {
	broker = &Broker{
		Notifier:       make(chan string, 1),
		newClients:     make(chan chan string),
		closingClients: make(chan chan string),
		clients:        make(map[chan string]struct{}),
	}

	go broker.listen()

	return
}

func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:
			broker.clients[s] = struct{}{}
		case s := <-broker.closingClients:
			delete(broker.clients, s)
		case event := <-broker.Notifier:
			for clientMessageChan, _ := range broker.clients {
				clientMessageChan <- event
			}
		}
	}
}
