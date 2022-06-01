### Concept:

gosquito is a relatively lightweight tool for fetching/preprocessing data at the edge. It doesn't intend to be a comprehensive tool for everything, instead it uses easy-to-use plugins primitives for base operation over data and transmitting produced data further.

### Basic workflow:

1. Input plugin receives data and forming data items.
2. Datums are usual structs with fields.
3. Process plugins use datums as a data source (input plugins fields) and as destination for saving data (data and iter fields).
4. Process plugins don't make copies of input plugin data, they always work with the same set of datums (through pointers), but different process plugins use different data fields for saving data.
5. Process plugins can reuse results of other process plugins through "require" option. Those results are not copies, but always the same set of original datums.
6. Process plugins should explicitly include data (include option) for sending data with output plugin.
7. Output plugin sends data to destinations.


#### Datum structure:

```go
type Datum struct {
  FLOW       string           // Flow name.
  PLUGIN     string           // Input plugin name (rss, twitter, telegram etc.).
  SOURCE     string           // Input plugin source (feed url, twitter channel, telegram chat etc.).
  TIME       time.Time        // Time of article, tweet, message.
  TIMEFORMAT string           // User defined time format (common field).
  TIMEZONE   time.Location    // User defined timezone (common field).
  UUID       uuid.UUID        // "Unique" id of data item.
  
  DATA       Data             // Temporary structure for keeping static data.
  ITER       Iter             // Temporary structure for keeping iterated data.
  
  RESTY      Resty            // Resty plugin structure.
  RSS        Rss              // RSS plugin structure.
  TELEGRAM   Telegram         // Telegram plugin structure.
  TWITTER    Twitter          // Twitter plugin structure.
  
  WARNINGS   []string         // Contains plugins' warnings.
}
```

#### Structure for keeping data:

```go
type Data struct {
  ARRAY0 []string
  ARRAY1 []string
  ARRAY2 []string
  ARRAY3 []string
  ARRAY4 []string
  ARRAY5 []string
  ARRAY6 []string
  ARRAY7 []string
  ARRAY8 []string
  ARRAY9 []string
  ARRAY10 []string
  ARRAY11 []string
  ARRAY12 []string
  ARRAY13 []string
  ARRAY14 []string
  ARRAY15 []string

  ARRAYA []string
  ARRAYB []string
  ARRAYC []string
  ARRAYD []string
  ARRAYE []string
  ARRAYF []string
  ARRAYG []string
  ARRAYH []string
  ARRAYI []string
  ARRAYJ []string
  ARRAYK []string
  ARRAYL []string
  ARRAYM []string
  ARRAYN []string
  ARRAYO []string
  ARRAYP []string

  TEXT0  string
  TEXT1  string
  TEXT2  string
  TEXT3  string
  TEXT4  string
  TEXT5  string
  TEXT6  string
  TEXT7  string
  TEXT8  string
  TEXT9  string
  TEXT10 string
  TEXT11 string
  TEXT12 string
  TEXT13 string
  TEXT14 string
  TEXT15 string
  TEXT16 string
  TEXT17 string
  TEXT18 string
  TEXT19 string
  TEXT20 string
  TEXT21 string
  TEXT22 string
  TEXT23 string
  TEXT24 string
  TEXT25 string
  
  TEXTA string
  TEXTB string
  TEXTC string
  TEXTD string
  TEXTE string
  TEXTF string
  TEXTG string
  TEXTH string
  TEXTI string
  TEXTJ string
  TEXTK string
  TEXTL string
  TEXTM string
  TEXTN string
  TEXTO string
  TEXTP string
  TEXTQ string
  TEXTR string
  TEXTS string
  TEXTT string
  TEXTU string
  TEXTV string
  TEXTW string
  TEXTX string
  TEXTY string
  TEXTZ string
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
