### Storage:

gosquito uses [badger](https://github.com/dgraph-io/badger) (key/value storage) as a storage for keeping flow states.  

### Base workflow:  

1. Input plugin receives data.
2. If match_signature is not specified (default value), gosquito compares input data timestamps with saved per source timestamps.  Example: <SOURCE\>:\<TIMESTAMP\>.
3. If match_signature is specified (various by input plugins), gosquito generate hash for specific fields and checks it existing in database (if hash has been found - it's not a new data).  Example: SHA(\<SOURCE\>\<FIELD1\>...\<FIELDN\>).
