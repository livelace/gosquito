### Data model:

1. Every plugin works with DataItem.
2. DataItem - it's a data structure with specific fields.
3. There are common data fields shared by all plugins and plugin specific data fields.

Data item structure.

```go
type DataItem struct {
	FLOW       string      // flow name, common field.
	LANG       string      // data item language, common field.
	PLUGIN     string      // plugin name, plugin specific field.
	SOURCE     string      // plugin source, plugin specific field.
	TIME       time.Time   
	TIMEFORMAT string
	TIMEZONE   *time.Location
	UUID       uuid.UUID

	DATA     Data
	RSS      RssData
	TELEGRAM TelegramData
	TWITTER  TwitterData
}
```