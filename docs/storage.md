### Storage:

gosquito uses [badger](https://github.com/dgraph-io/badger) (key/value storage) as a storage for keeping flow states.  

### Base workflow:  

1. Input plugin receives data.
2. If match_signature is not specified (default value), gosquito compares input data timestamp with saved per source timestamp.<br>Database record: <SOURCE\>:\<TIMESTAMP\>.
3. If match_signature is specified (various for input plugins), gosquito generate hash for specific fields and checks it existing in database (if hash has been found - it's not a new data).<br>Database record: SHA1(\<SOURCE\>\<FIELD1\>...\<FIELDN\>):\<TIMESTAMP\>.
4. Every save record has [TTL](https://dgraph.io/docs/badger/get-started/#setting-time-to-live-ttl-and-user-metadata-on-keys) (match_ttl option, 1 day by default). Data will be considered as new after record expiration.