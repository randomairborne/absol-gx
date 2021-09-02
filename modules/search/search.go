package search

import (
	"github.com/bwmarrin/discordgo"
	"github.com/lordralex/absol/api"
	"github.com/lordralex/absol/api/database"
	"github.com/lordralex/absol/api/logger"
	"github.com/lordralex/absol/modules/factoids"
	"github.com/spf13/viper"
	"strconv"
	"strings"
)

type Module struct {
	api.Module
}

// Load absol commands API
func (*Module) Load(ds *discordgo.Session) {
	api.RegisterCommand("search", RunCommand)

	api.RegisterIntentNeed(discordgo.IntentsGuildMessages, discordgo.IntentsDirectMessages)
}

func RunCommand(ds *discordgo.Session, mc *discordgo.MessageCreate, _ string, args []string) {
	if mc.GuildID != "" {
		_ = factoids.SendWithSelfDelete(ds, mc.ChannelID, "This command may only be used in DMs.")
		return
	}

	if len(args) == 0 {
		_, _ = ds.ChannelMessageSend(mc.ChannelID, "You must specify a search string!")
		return
	} else if len(strings.Join(args, "")) < 3 {
		_, _ = ds.ChannelMessageSend(mc.ChannelID, "Your search is too short!")
		return
	}

	max := viper.GetInt("factoids.max")
	if max == 0 {
		max = 5
	}

	db, err := database.Get()
	if err != nil {
		_, _ = ds.ChannelMessageSend(mc.ChannelID, "Failed to connect to database")
		logger.Err().Printf("Failed to connect to database\n%s", err)
		return
	}

	pageNumber := 0
	pageNumber, err = strconv.Atoi(args[len(args)-1]) // if the last arg is a number use it as the page number

	// if the page number was specified then we subtract one from it to make the page index start at 1, then
	// cut the last argument out if it's a number
	if _, err := strconv.Atoi(args[len(args)-1]); err == nil {
		pageNumber = pageNumber - 1
		args = args[:len(args)-1]
	}

	message := ""
	// gets how many rows there are
	var rows int64
	db.Where("content LIKE ? OR name LIKE ?", "%"+strings.Join(args, " ")+"%", "%"+strings.Join(args, " ")+"%").Table("factoids").Count(&rows)

	// ensures that page number is valid
	if pageNumber < 0 || pageNumber > int(rows)/max+1 {
		pageNumber = 0
	}

	// searches through results for a match
	// gets the factoids table
	var factoidsList []factoids.Factoid
	db.Where("content LIKE ? OR name LIKE ?", "%"+strings.Join(args, " ")+"%", "%"+strings.Join(args, " ")+"%").Order("name").Offset(pageNumber * max).Limit(max).Find(&factoidsList)
	// actually put the factoids into one string
	for _, factoid := range factoidsList {
		message += "**" + factoid.Name + "**" + "```" + factoids.CleanupFactoid(factoid.Content) + "```\n"
	}

	// add footer with page numbers
	footer := ""
	footer = "Page " + strconv.Itoa(pageNumber+1) + "/" + strconv.Itoa(int(rows)/max+1) + ". "
	if pageNumber+1 < int(rows)/max+1 {
		footer += "Type !?search " + strings.Join(args, " ") + " " + strconv.Itoa(pageNumber+2) + " to see the next page."
	}

	// prepare embed
	embed := &discordgo.MessageEmbed{
		Description: message,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footer,
		},
	}

	send := &discordgo.MessageSend{
		Embed: embed,
	}

	_, err = ds.ChannelMessageSendComplex(mc.ChannelID, send)
	if err != nil {
		logger.Err().Printf("Failed to send message\n%s", err)
	}

}
