### Data processing:

Typical flow steps:

1. **Input plugin** gathers data and produces set of DataItem.
4. **Process plugins** handle that DataItem set and fill chosen fields with produced data.
5. Finally, **Output plugin** constructs messages with selected DataItem fields. 


DataItem is a data structure with specific fields, contains common for all plugins fields and plugin specific fields.

```go
type DataItem struct {
	FLOW       string           // Flow name (common field).
	LANG       string           // Input or process plugins can fill this field.
	PLUGIN     string           // Input plugin name (rss, twitter, telegram etc.).
	SOURCE     string           // Input plugin source (feed, channel, chat etc.).
	TIME       time.Time        // Time of article, tweet, message.
	TIMEFORMAT string           // User defined time format (common field).
	TIMEZONE   *time.Location   // User defined timezone (common field).
	UUID       uuid.UUID        // "Unique" id of data item.

	DATA     Data               // Temporary structure for keeping process plugins results.
	RSS      RssData            // Contains nested RSS plugin structure.
	TELEGRAM TelegramData       // Contains nested Telegram plugin structure.
	TWITTER  TwitterData        // Contains nested Twitter plugin structure.
}
```
