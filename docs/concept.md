### Concept:

gosquito is a relatively lightweight tool for fetching/preprocessing data at the edge. It doesn't intend to be a comprehensive tool for everything, instead it uses easy-to-use plugins primitives for base operation over data and transmitting produced data further.

### Basic workflow:

1. Input plugin receives data and forming data items.
2. Data items are usual structs with fields.
3. Process plugins use data items as a data source (input plugins fields) and as destination for saving data (data and iter fields).
4. Process plugins don't make copies of input plugin data, they always work with the same set of data items (through pointers), but different process plugins use different data fields for saving data.
5. Process plugins can reuse results of other process plugins through "require" option. Those results are not copies, but always the same set of original data items.
6. Process plugins should explicitly include data (include option) for sending data with output plugin.
7. Output plugin sends data to destinations.


#### Data item structure:

```go
type DataItem struct {
	FLOW       string           // Flow name.
	PLUGIN     string           // Input plugin name (rss, twitter, telegram etc.).
	SOURCE     string           // Input plugin source (feed url, twitter channel, telegram chat etc.).
	TIME       time.Time        // Time of article, tweet, message.
	TIMEFORMAT string           // User defined time format (common field).
	TIMEZONE   *time.Location   // User defined timezone (common field).
	UUID       uuid.UUID        // "Unique" id of data item.

	DATA     Data               // Temporary structure for keeping static data.
	ITER     Iter               // Temporary structure for keeping iterated data.
	
	RESTY    Resty              // Resty plugin structure.
	RSS      Rss                // RSS plugin structure.
	TELEGRAM Telegram           // Telegram plugin structure.
	TWITTER  Twitter            // Twitter plugin structure.
}
```

#### Structure for keeping static data:

```go
type Data struct {
	ARRAY0 []string
	ARRAY9 []string
	TEXT0  string
	TEXT9  string
}
```

#### Structure for keeping iterated data:

```go
type Iter struct {
	INDEX int
	VALUE string
}
```

#### Plugin specific data structures:

1. [RESTY](plugins/input/resty.md)    
2. [RSS](plugins/input/rss.md)  
3. [TELEGRAM](plugins/input/telegram.md)  
4. [TWITTER](plugins/input/twitter.md)  
