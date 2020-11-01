### Data processing:

Typical flow steps:

1. *Input plugin* gathers data and produces set of *DataItem*.
2. *Process plugins* handle *DataItem* set and fill chosen fields with
   produced data.
3. Finally, *Output plugin* constructs messages with selected *DataItem*
   fields and send messages to destinations.


*DataItem* is a data structure with specific fields, contains common
for all plugins fields and plugin specific fields (typical flow
consists of plugins with *DataItem* fields as parameters.

```go
type DataItem struct {
	FLOW       string           // Flow name (common field).
	PLUGIN     string           // Input plugin name (rss, twitter, telegram etc.).
	SOURCE     string           // Input plugin source (feed url, twitter channel, telegram chat etc.).
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
	ARRAY9 []string

	TEXT0   string
        // ...
	TEXT9  string
}
```

Plugin specific data structures:
[RSS](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/rss.md),
[Telegram](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/telegram.md),
[Twitter](https://github.com/livelace/gosquito/blob/master/docs/plugins/input/twitter.md)
