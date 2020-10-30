### Data processing:

Typical flow steps:

1. **Input plugin** gathers data and produces set of DataItem.
2. **Process plugins** handle that DataItem set and fill chosen fields with produced data.
3. Finally, **Output plugin** constructs messages with selected DataItem fields and send them to destinations. 


DataItem is a data structure with specific fields, contains common for all plugins fields and plugin specific fields.  
Typical flow configuration consists of plugins and DataItem fields as parameters.

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

Plugin specific data structures: [RSS](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/rss.md), [Telegram](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/telegram.md), [Twitter](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/twitter.md)

DataItem.Data structure:
```go
type Data struct {
	ARRAY0  []string
	ARRAY1  []string
	ARRAY2  []string
	ARRAY3  []string
	ARRAY4  []string
	ARRAY5  []string
	ARRAY6  []string
	ARRAY7  []string
	ARRAY8  []string
	ARRAY9  []string
	ARRAY10 []string
	TEXT0   string
	TEXT1   string
	TEXT2   string
	TEXT3   string
	TEXT4   string
	TEXT5   string
	TEXT6   string
	TEXT7   string
	TEXT8   string
	TEXT9   string
	TEXT10  string
}
```
