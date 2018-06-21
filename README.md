# BullBot
Simple discord image bot in go
# Usage
```
./bullbot --token=<Discord bot token>
```
First, open sqlite browser on the generated config.db and execute
```
INSERT INTO ranks (userid, rank) VALUES ('YOUR-DISCORD#1234', 10)
```
Then, in DM:

`!admin <user#1234> [user#5678] […]` toggles admin status for the given user(s). Allows adding and removing images.

`<category> <url> [url] […]` add given image(s) to `category`

`!remove <messageID> [messageID] […]` removes image(s) contained in given message(s).
`messageID` may be the ID of either the message adding the image(s) or any image message sent by the bot.

Removing images also works by deleting the original message sent to add the image (i.e. the one sent by admin in dm)
# Building
```
go build github.com/bullcheat/bullbot -ldflags="-s -w" # Cuts binary size by half
```
It is highly recommended to compress the binary executable with `upx` as this will cut the size by another 2/3.
```
upx bullbot
```
