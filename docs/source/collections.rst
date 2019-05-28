Private Data Collections
========================

Transient Data Collection
-------------------------

A transient data collection is a **private data collection** that holds temporary data off of the ledger (i.e. it is stored at endorsement time) and therefore does not require a commit.

Transient data should have an *expiration based on time* (as opposed to block height) and the expiration should be assumed to be (typically) short, in which case in-memory store is suitable.
In order to prevent too much data from being stored in memory, the data should be off-loaded to a database using an LRU policy.
In order to make storing transient data efficient, transient data should not be stored on every peer but should be distributed to a small subset of peers (calculated using the collection config) according to a deterministic algorithm.

When a client requests the data, then the same algorithm may be used to find the peer in which it was stored. The existing chaincode stub functions, PutPrivateData and GetPrivateData, may be used by chaincode to put and get transient data. Reads and writes of transient data must not show up in the read/write set if the transaction is committed to the ledger.
An API should also be provided in order to read/write transient data from outside of a chaincode invocation.

Off-Ledger and DCAS Collections
-------------------------------

An Off-Ledger collection is a **private data collection** that stores data at endorsement time, i.e. the transaction does not need to be committed.
Purging of Off-Ledger data should be based on a time-to-live policy as opposed to a block-to-live policy. Reads and writes of Off-Ledger/DCAS data must not show up in the read/write set if the transaction is committed to the ledger.
Aside from the above requirements, Off-Ledger private collections behave the same way as standard private data collections.

A **DCAS** (Distributed Content Addressable Store) collection is a subset of the Off-Ledger collection with the restriction that the key MUST be the hash of the value.
The existing chaincode stub functions, PutPrivateData and GetPrivateData, may be used by chaincode to put and get Off-Ledger/DCAS data.
For **DCAS collections**, the key may be empty ("") since it is always the hash of the value.
If the key is provided, then it should validated against the value. API's should also be provided for both Off-Ledger and DCAS collections in order to read/write data from outside of a chaincode invocation.

.. Licensed under the Apache License, Version 2.0 (Apache-2.0)
https://www.apache.org/licenses/LICENSE-2.0
