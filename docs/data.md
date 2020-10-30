### Data processing:

Typical flow steps:

1. **Input plugin** gathers data and produces set of **DataItem**.
2. **Process plugins** handle that **DataItem** set and fill chosen fields with produced data.
3. Finally, **Output plugin** constructs messages with selected **DataItem** fields and send them to destinations. 


**DataItem** is a data structure with specific fields, contains common for all plugins fields and plugin specific fields.  
Typical flow configuration consists of plugins with **DataItem** fields as parameters.

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

Temporary **DataItem.Data** structure for process plugins results:
```go
type Data struct {
	ARRAY0  []string
        // ...
	ARRAY10 []string

	TEXT0   string
        // ...
	TEXT10  string
}
```

Plugin specific data structures: [DataItem.RSS](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/rss.md), [DataItem.Telegram](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/telegram.md), [DataItem.Twitter](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/twitter.md)