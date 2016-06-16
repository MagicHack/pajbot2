package boss

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"strings"
	"sync"

	"github.com/pajlada/pajbot2/redismanager"
	"github.com/pajlada/pajbot2/sqlmanager"

	"github.com/pajlada/pajbot2/bot"
	"github.com/pajlada/pajbot2/common"
	"github.com/pajlada/pajbot2/modules"
)

/*
The Irc object contains all data xD
*/
type Irc struct {
	sync.Mutex
	server   string
	port     string
	pass     string
	nick     string
	conn     net.Conn
	ReadChan chan string
	SendChan chan string
	bots     map[string]chan common.Msg
	redis    *redismanager.RedisManager
	sql      *sqlmanager.SQLManager
	parser   *parse
	quit     chan string
}

/*
SendRaw sends a raw message to the given connection.
The only thing it appends is \r\n
*/
func (irc *Irc) SendRaw(s net.Conn, line string) {
	fmt.Fprint(s, line+"\r\n")
}

func (irc *Irc) newConn() {
	if irc.conn != nil {
		return
	}
	conn, err := net.Dial("tcp", irc.server+":"+irc.port)
	if err != nil {
		fmt.Println("Error connecting to the IRC servers:", err)
		return
	}
	if irc.pass != "" {
		irc.SendRaw(conn, "PASS "+irc.pass)
	}
	irc.SendRaw(conn, "NICK "+irc.nick)
	go irc.readConnection(conn)
	irc.conn = conn
	fmt.Println("connected")
}

func (irc *Irc) send() {
	for {
		msg := <-irc.SendChan
		irc.SendRaw(irc.conn, msg)
		fmt.Println("sent: " + msg)
	}
}

// GetGlobalUser fills in the global user in the message from redis
func (irc *Irc) GetGlobalUser(m *common.Msg) {
	u := &common.GlobalUser{}
	irc.redis.GetGlobalUser(m.Channel, &m.User, u)
	if m.Type == common.MsgWhisper {
		m.Channel = u.Channel
	}
}

func (irc *Irc) readConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)
	readChan := make(chan string)
	running := true
	go func() {
		var line string
		for running {
			line = <-readChan
			if strings.HasPrefix(line, "PING") {
				irc.SendRaw(conn, strings.Replace(line, "PING", "PONG", 1))
			} else {
				m := irc.parser.Parse(line)
				// throw away its own and other useless msgs
				if m.User.Name == irc.nick {
					// Throw away its own messages
					continue
				}
				log.Println(m.Type)
				switch m.Type {
				case common.MsgPrivmsg, common.MsgWhisper:
					irc.GetGlobalUser(&m)
					if m.Channel != "" {
						irc.bots[m.Channel] <- m
					} else {
						log.Println("No channel for message")
					}
				case common.MsgThrowAway:
					// Do nothing
					break
				default:
					log.Printf("Unhandled message[%d]: %s\n", m.Type, m.Message)
				}
			}
		}
	}()
	defer func() {
		running = false
		close(readChan)
	}()
	for {
		line, err := tp.ReadLine()
		if err != nil {
			log.Println("connection died", err)
			irc.newConn()
			//irc.JoinChannels(irc.readConn[conn])
			return
		}
		readChan <- line
	}
}

// NewBot creates a new bot in the given channel
func (irc *Irc) NewBot(channel string) {
	read := make(chan common.Msg)
	newbot := bot.Config{
		Quit:     irc.quit,
		Channel:  channel,
		ReadChan: read,
		SendChan: irc.SendChan,
		Redis:    irc.redis,
	}
	irc.bots[channel] = read
	commandModule := &modules.Command{}
	// TODO: This should be generalized (and optional if possible)
	// Could that be based on module type?
	// If module.@type == 'NeedsInit' { (cast)module.Init() }
	commandModule.Init(irc.sql)
	_modules := []bot.Module{
		&modules.Banphrase{},
		commandModule,
		&modules.Pyramid{},
		&modules.Quit{},
	}
	b := bot.NewBot(newbot, _modules)
	go b.Init()
}

/*
JoinChannel joins a twitch chat and creates a new bot if there isnt already one
*/
func (irc *Irc) JoinChannel(channel string) {
	irc.Lock()
	defer irc.Unlock()
	if _, ok := irc.bots[channel]; !ok {
		irc.NewBot(channel)
		irc.SendRaw(irc.conn, "JOIN #"+channel)
	}
}

/*
JoinChannels joins a list of channels, given as a string slice
*/
func (irc *Irc) JoinChannels(channels []string) {
	for _, channel := range channels {
		irc.JoinChannel(channel)
	}
}

/*
Init initalizes shit.

TODO: This should just create the Irc object. You should have to call
irc.Run() manually I think. or irc.Start()?
*/
func Init(config *common.Config) *Irc {
	server := "irc.chat.twitch.tv"
	port := "80"
	//usingBroker := false
	if config.BrokerPort != "" {
		server = "localhost"
		port = config.BrokerPort
		//usingBroker = true
	}
	irc := &Irc{
		server:   server,
		port:     port,
		pass:     config.Pass,
		nick:     config.Nick,
		ReadChan: make(chan string, 10),
		SendChan: make(chan string, 10),
		bots:     make(map[string]chan common.Msg),
		redis:    redismanager.Init(config),
		sql:      sqlmanager.Init(config),
		parser:   &parse{},
		quit:     config.Quit,
	}
	irc.newConn()
	go irc.send()
	go irc.JoinChannels(config.Channels)
	return irc
}
