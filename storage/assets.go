package storage

import (
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/saeveritt/go-peerassets/protobuf"
	"github.com/saeveritt/go-peerassets/utils"
	ppcd "github.com/saeveritt/go-peercoind"
	"log"
	"strconv"
)

var(
	//add to subscribed map the list of deck id's you wish to import
	subscribed = map[string]bool{
		"*": true,
	}
)

func PutRootAsset(cards bool){
	// Loads all valid assets registered to main p2th address registry
	txs := utils.RootTransactions()
	rawtxs := utils.RawTransactions(txs)
	i := 0 // Deck counter
	for _, rawtx := range rawtxs{
		if _,ok := subscribed["*"];!ok{continue}
		if _,ok := subscribed[rawtx.Txid];!ok || subscribed["*"] {
			sender := utils.GetSender(rawtx)
			receiver := utils.GetReceiver(rawtx)
			opReturn := utils.GetMetaData(rawtx)
			deck := utils.DeckParse(opReturn)
			err := utils.ValidateDeckBasic(receiver, deck)
			if err != nil {
				//log.Print(err)
				continue
			}
			proto, err := deck.XXX_Marshal(nil, false)
			must(err)
			if sender != "coinbase/coinstake" && len(proto) != 0 {
				PutDeck(sender, rawtx)
				PutDeckProto(proto, rawtx)
				PutDeckCreator(sender, rawtx, proto)
				if cards {utils.ImportDeck(rawtx.Txid)}
				i++
				fmt.Printf("\r%d Decks Validated", i)
			}
		}
	}
}

func PutDeck(sender string, rawtx ppcd.RawTransaction){
	//Import deck information into local db
	utils.ImportDeck(rawtx.Txid)
	// Bucket: Decks, Key: DeckSpawn ID, Value: Deck Owner
	Put("Decks",rawtx.Txid,[]byte(sender))
}
func PutDeckProto(proto []byte, rawtx ppcd.RawTransaction){
	//Import deck information into local db
	utils.ImportDeck(rawtx.Txid)
	// Bucket: Decks, Key: DeckSpawn ID, Value: Deck Owner
	Put("DecksProto",rawtx.Txid, proto)
}
func PutDeckCreator(sender string, rawtx ppcd.RawTransaction,proto []byte){
	// Bucket: <sender address>, Key: "Deck-" + <Deckspawn ID>, Value: <proto>
	Put(sender,"Deck-" + rawtx.Txid,proto)
}


func PutAllCards(){
	db, _:=Connect()
	db.View(func(tx *bolt.Tx) error{
		tx.Bucket([]byte("Decks")).ForEach( func(k ,v []byte) error{
			deckid := string(k)
			log.Print("Searching Buckets" + deckid)
			log.Print(deckid)
			utils.ImportDeck(deckid)
			PutCards(deckid)
			return nil
		} )
		return nil
		})
	db.Close()
}

func PutCards(deckid string){
	// Loads all valid assets registered to main p2th address registry
	cards := utils.GetCards(deckid)
	for _, card := range cards{
		log.Print(card)
		ProcessDeckCardKeys(card)
	}
}

func ProcessDeckCardKeys(card *protobuf.CardTransfer){
	height:= strconv.Itoa( int(card.BlockHeight[0]) )
	txIndex := strconv.Itoa( int(card.TxIndex[0]) )
	cardIndex := strconv.Itoa( int(card.CardIndex[0]) )
	baseKey := card.DeckId + "-" + height + "-" + txIndex + "-" + cardIndex
	sendKey := "Card-Send-"+ baseKey
	receiveKey := "Card-Receive-" + baseKey
	proto,_ := card.XXX_Marshal(nil,false)
	Put(card.Sender,sendKey, proto)
	Put(card.Receiver[0],receiveKey, proto)
}