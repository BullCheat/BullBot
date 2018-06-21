package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	_ "github.com/mattn/go-sqlite3"
	"github.com/bwmarrin/discordgo"
	"regexp"
	"database/sql"
	"strings"
	"bytes"
	"strconv"
	"net/url"
)

var (
	Token string
	DBFile string
	matcher, _ = regexp.Compile("!\\S+")
	db *sql.DB
)

func init() {
	flag.StringVar(&Token, "token", "", "Bot Token")
	flag.StringVar(&DBFile, "file", "config.db", "Fichier de sauvegarde")
	flag.Parse()
	if Token == "" {
		println("BullBot.exe -token <token>")
	}
}

func main() {
	initDB()
	initDiscord()
}
func initDiscord() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		println("Erreur lors de la création de la session discord :", err)
		return
	}
	dg.AddHandler(messageCreate)
	dg.AddHandler(messageDelete)
	err = dg.Open()
	if err != nil {
		println("Erreur lors de l'ouverture de la connexion", err)
		return
	}

	println("Le bot a démarré. CTRL + C pour fermer.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}
func initDB() {
	dir, e := os.Getwd()
	if e != nil {
		panic(e.Error())
	}
	var err error
	println(dir)
	db, err = sql.Open("sqlite3", "file:" + DBFile + "?cache=shared")
	if err != nil {
		panic(err.Error())
	}
	db.SetMaxOpenConns(1)
	db.Exec("CREATE TABLE IF NOT EXISTS images	 (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, url TEXT)")
	db.Exec("CREATE TABLE IF NOT EXISTS ranks (id INTEGER PRIMARY KEY AUTOINCREMENT, userid TEXT, rank INTEGER)")
	db.Exec("CREATE TABLE IF NOT EXISTS history (messageid TEXT, image INT NOT NULL, CONSTRAINT history_images_id_fk FOREIGN KEY (image) REFERENCES images (id) ON DELETE CASCADE ON UPDATE CASCADE)")
}

func messageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	channel, e := s.Channel(m.ChannelID)
	if e != nil {
		println(e.Error())
		return
	}
	if channel.Type == discordgo.ChannelTypeDM {
		i := deleteImage(m.ID)
		if i > 0 {
			s.ChannelMessageSend(m.ChannelID, strconv.Itoa(i) + " images supprimées.")
		}
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	channel, e := s.Channel(m.ChannelID)
	if e != nil {
		println(e.Error())
		return
	}
	if channel.Type == discordgo.ChannelTypeDM {
		tryAdding(s, channel, m.Message)
	} else {
		react(s, channel, m.Message)
	}
}
func tryAdding(s *discordgo.Session, channel *discordgo.Channel, message *discordgo.Message) {
	userid := message.Author.Username + "#" + message.Author.Discriminator
	row := db.QueryRow("SELECT IFNULL((SELECT rank FROM ranks WHERE userid = ?), 0)", userid)
	var rank int
	err := row.Scan(&rank)
	if err != nil {
		println(err.Error())
		return
	}
	if rank < 10 {
		s.ChannelMessageSend(channel.ID, "Veuillez demander à l'administrateur du bot d'effectuer la commande suivante : `!admin " + userid + "`")
	} else {
		content := strings.Split(message.Content, " ")
		if strings.HasPrefix(content[0], "!") {
			command := content[0][1:]
			if command == "admin" {
				if len(content) == 1 {
					s.ChannelMessageSend(channel.ID, "!admin <userid1> [userid2…]")
				} else {
					actions :=  make([]bool, len(content)-1)
					for i := 1; i < len(content); i++ {
						var err error
						if err != nil {
							continue
						}
						res := db.QueryRow("SELECT EXISTS(SELECT id FROM ranks WHERE userid = ?)", content[i])
						var exists bool
						err = res.Scan(&exists)
						if err != nil {
							println(err.Error())
							return
						}
						actions[i-1] = !exists
						if exists {
							_, err = db.Exec("DELETE FROM ranks WHERE userid = ?", content[i])
							if err != nil {
								println("X " + err.Error())
							}
						} else {
							_, err =db.Exec(
								"INSERT INTO ranks (userid, rank) VALUES (?, ?)",
								content[i], 10)
							if err != nil {
								println("Y " + err.Error())
							}
						}
					}
					var b bytes.Buffer
					for i := 1; i < len(content); i++ {
						b.WriteString(content[i])
						b.WriteString(" : ")
						if actions[i-1] {
							b.WriteString("ajouté")
						} else {
							b.WriteString("retiré")
						}
						b.WriteString(" administrateur.")
					}
					s.ChannelMessageSend(channel.ID, "```" + b.String() + "```")
				}
			} else if command == "delete" {
				if len(content) == 1 {
					s.ChannelMessageSend(channel.ID, "!delete <id du message où apparait l'image> […]")
				} else {
					num := 0
					for i := 1; i < len(content); i++{
						num += deleteImage(content[i])
					}
					s.ChannelMessageSend(channel.ID, strconv.Itoa(num) + " images supprimées.")
				}
			} else {
				s.ChannelMessageSend(channel.ID, "Commande inconnue")
			}
		} else {
			if len(content) == 0 {
				s.ChannelMessageSend(channel.ID, "<command> <url1> [url2…]")
			} else if len(content) == 1 {
				s.ChannelMessageSend(channel.ID, content[0] + " <url1> [url2…]")
			} else {
				num := 0
				for i := 1; i < len(content); i++ {
					u := content[i]
					_, err = url.ParseRequestURI(u)
					if err != nil {
						s.ChannelMessageSend(channel.ID, "Image n°" + strconv.Itoa(i) + " invalide")
					} else {
						res, _ := db.Exec("INSERT INTO images(name, url) VALUES (?, ?)", strings.ToLower(content[0]), u, message.ID)
						db.Exec("INSERT INTO history(messageid, image) VALUES (?, last_insert_rowid())", message.ID)
						n, _ := res.RowsAffected()
						num += int(n)
					}
				}
				s.ChannelMessageSend(channel.ID, strconv.Itoa(num) + " images ajoutées.")
			}
		}
	}
}
func react(s *discordgo.Session, channel *discordgo.Channel, message *discordgo.Message) {
	matches := matcher.FindAllString(message.Content, -1)
	if len(matches) == 0 {
		return
	}
	for i := range matches {
		match := matches[i][1:]
		//https://stackoverflow.com/questions/4114940/select-random-rows-in-sqlite
		row := db.QueryRow(
			"SELECT IFNULL((SELECT id || ' ' || url FROM images WHERE id IN (SELECT id FROM images WHERE name = ? ORDER BY RANDOM() LIMIT 1)), '')",
			strings.ToLower(match))
		var ret string
		err := row.Scan(&ret)

		if err != nil {
			println(err.Error())
			continue
		}

		if ret != "" {
			arr := strings.Split(ret, " ")
			id := arr[0]
			link := strings.Join(arr[1:], " ")
			embed := discordgo.MessageEmbed{Image:&discordgo.MessageEmbedImage{URL:link}}
			msg, err := s.ChannelMessageSendEmbed(channel.ID, &embed)
			if err != nil {
				println(err)
			} else {
				db.Exec("INSERT INTO history(messageid, image) VALUES (?, ?)", msg.ID, id)
			}
		}
	}
}

func deleteImage(id string) int {
	res1 := db.QueryRow("SELECT image FROM history WHERE messageid = ?", id)
	var imageid int
	err := res1.Scan(&imageid)
	if err != nil {
		println("A " + err.Error())
		return 0
	}
	r, err := db.Exec("DELETE FROM images WHERE id = ?", imageid)
	if err != nil {
		println("C " + err.Error())
		return 0
	}
	i, err := r.RowsAffected()
	if err != nil {
		println("D " + err.Error())
		return 0
	}
	return int(i)
}