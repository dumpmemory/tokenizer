package tokenizer

// tokenizer represents a tokenization pipeline
// TODO: full description

import (
	// "bufio"
	// "context"
	// "fmt"
	// "log"
	// "math"
	// "os"
	// "reflect"
	// "regexp"
	// "strings"
	// "sync"

	// progressbar "github.com/schollz/progressbar/v2"
	// "golang.org/x/sync/errgroup"

	"github.com/sugarme/tokenizer/normalizer"
	// "github.com/sugarme/tokenizer/util"
)

const mb = 1024 * 1024
const gb = 1024 * mb

type Offsets struct {
	Start int
	End   int
}

type PreToken struct {
	Value   string
	Offsets Offsets
}

type Token struct {
	Id      int
	Value   string
	Offsets Offsets
}

// PreTokenizer processes strings before going to the model
// It splits the given string into multiple substrings and keeps track
// of offsets of split substrings from the `NormalizedString`. In some
// occasion, the `PreTokenizer` might need to modify the given `NormalizedString`
// to ensure it entirely keeps track of the offsets and the mapping with
// the original string.
type PreTokenizer interface {
	// PreTokenize(pretokenized PreTokenizedString) (retVal []PreToken)
	PreTokenize(pretokenized PreTokenizedString) (retVal PreTokenizedString, err error)
}

// Model represents a model used during tokenization (i.e., BPE, Word, or Unigram)
type Model interface {
	// Tokenize tokenizes the given sequence into multiple underlying `Token`
	// The `offsets` on the `Token` are expected to be relative to the given
	// sequence
	// Tokenize(tokens []PreToken) ([]Token, error)
	Tokenize(sequence string) ([]Token, error)
	// TokenToId finds the ID associated with a string token
	TokenToId(token string) (id int, ok bool)
	// IdToToken find the string token associated with an ID
	IdToToken(id int) (token string, ok bool)
	// GetVocab retrieves the entire vocabulary mapping (token -> Id)
	GetVocab() map[string]int
	// GetVocabSize retrieves the entire vocabulary mapping(map[token]id)
	GetVocabSize() int
	// Save saves the current `Model` in the given folder, using the
	// given `prefixOpt` for various files that need to be saved.
	Save(path string, prefixOpt ...string) error
}

// PostProcessor is in charge of post-processing an encoded output of
// the `Tokenizer`.
// It adds any special tokens that a language model would require.
type PostProcessor interface {
	// AddedTokens returns the number of tokens that will be added during the processing step
	AddedTokens(isPair bool) int
	// Process processes both encodings and returns a new merged one
	// NOTE: pairEncoding is optional
	Process(encoding Encoding, addSpecialTokens bool, encodingOpt ...Encoding) Encoding
}

// DefaultProcess is a helper function of PostProcessor's Process method
// It helps to fast track by just merging encoding and its pair.
func DefaultProcess(encoding Encoding, encodingOpt ...Encoding) Encoding {
	if len(encodingOpt) > 0 {
		pairEncoding := encodingOpt[0]
		return encoding.MergeWith(pairEncoding)
	}

	return encoding
}

// Decoder takes care of (merges) the given slice of tokens to string
type Decoder interface {
	Decode(tokens []string) string
}

// Trainer is responsible for training a model. It takes lines/sentences
// and returns a tokenizer `Model` when done.
type Trainer interface {
	// Whether showing progress bar or not
	WithProgressBar() bool
	// Actual training method. It will return a trained model and
	// a list of `special tokens` to be added directly to the tokenizer
	// along with the model
	Train(words map[string]int) (Model, []AddedToken)
	// ProcessTokens processes a bunch of tokens and counts them as relevant
	ProcessTokens(words map[string]int, tokens []string)
}

// Implement methods for `Token`
// NewToken generate new token from input data
func NewToken(id int, value string, offsets Offsets) Token {
	return Token{
		Id:      id,
		Value:   value,
		Offsets: offsets,
	}
}

// InputSequence :
// ===============

type InputSequence interface{}

func NewInputSequence(s string) (retVal InputSequence) {
	// TODO. implement
	return
}

func NewInputSequenceFromPreTokenized(preTokenized []string) (retVal InputSequence) {
	// TODO. implement
	return
}

type Single struct {
	Sentence InputSequence
}
type Dual struct {
	Sentence InputSequence
	Pair     InputSequence
}

type EncodeInput interface{}

func NewSingleEncodeInput(sentence InputSequence) (retVal EncodeInput) {
	return Single{sentence}
}

func NewDualEncodeInput(sentence, pairSentence InputSequence) (retVal EncodeInput) {
	return Dual{sentence, pairSentence}
}

// Tokenizer represents a tokenization pipeline.
// It can implement any encoding or decoding of any text.
type Tokenizer struct {
	// Parts
	normalizer    *normalizer.Normalizer // optional
	preTokenizer  *PreTokenizer          // optional
	model         Model
	postProcessor *PostProcessor // optional
	decoder       *Decoder       // optional

	// Added vocabulary capability
	addedVocabulary AddedVocabulary

	// General processing parameters
	trunc   *TruncationParams // optional
	padding *PaddingParams    // optional
}

// Implementing methods for Tokenizer
func NewTokenizer(model Model) Tokenizer {
	return Tokenizer{
		normalizer:      nil,
		preTokenizer:    nil,
		model:           model,
		postProcessor:   nil,
		decoder:         nil,
		addedVocabulary: NewAddedVocabulary(),
		trunc:           nil,
		padding:         nil,
	}
}

func (t *Tokenizer) WithNormalizer(n normalizer.Normalizer) {
	t.normalizer = &n
}

func (t *Tokenizer) GetNormalizer() normalizer.Normalizer {
	return *t.normalizer
}

func (t *Tokenizer) WithPreTokenizer(preTokenizer PreTokenizer) {
	t.preTokenizer = &preTokenizer
}

func (t *Tokenizer) GetPreTokenizer() PreTokenizer {
	return *t.preTokenizer
}

func (t *Tokenizer) WithPostProcessor(postProcessor PostProcessor) {
	t.postProcessor = &postProcessor
}

func (t *Tokenizer) GetPostProcessor() PostProcessor {
	return *t.postProcessor
}

func (t *Tokenizer) WithDecoder(decoder Decoder) {
	t.decoder = &decoder
}

func (t *Tokenizer) GetDecoder() Decoder {
	return *t.decoder
}

func (t *Tokenizer) WithModel(model Model) {
	t.model = model
}

func (t *Tokenizer) GetModel() Model {
	return t.model
}

func (t *Tokenizer) WithTruncation(trunc TruncationParams) {
	t.trunc = &trunc
}

func (t *Tokenizer) GetTruncation() TruncationParams {
	return *t.trunc
}

func (t *Tokenizer) WithPadding(padding PaddingParams) {
	t.padding = &padding
}

// GetVocab get the vocabulary
func (t *Tokenizer) GetVocab(withAddedTokens bool) map[string]int {
	finalVocab := t.model.GetVocab()
	if withAddedTokens {
		addedVocab := t.addedVocabulary.GetVocab()
		if len(addedVocab) > 0 {
			for k, v := range addedVocab {
				finalVocab[k] = v
			}
		}
	}

	return finalVocab
}

// GetVocabSize get the size of vocabulary
func (t *Tokenizer) GetVocabSize(withAddedTokens bool) int {
	if !withAddedTokens {
		return t.model.GetVocabSize()
	}

	return t.model.GetVocabSize() + t.addedVocabulary.Len()
}

// TokenToId converts a token to a corresponding id
func (t *Tokenizer) TokenToId(token string) (id int, ok bool) {
	id, ok = t.addedVocabulary.TokenToId(token, t.model)
	return id, ok
}

// IdToToken converts an Id to a corresponding token
func (t *Tokenizer) IdToToken(id int) (token string, ok bool) {
	token, ok = t.addedVocabulary.IdToToken(id, t.model)
	return token, ok
}

/*
 * // Normalize normalizes the given sentence and return the corresponding normalized string
 * func (t *Tokenizer) Normalize(sentence string) (retVal normalizer.NormalizedString) {
 *
 *   isPairs := t.addedVocabulary.ExtractAndNormalize(sentence, t.normalizer)
 *   for _, isPair := range isPairs {
 *     if isPair.Id != nil { // id is optional
 *       return isPair.Substring.Normalized
 *     } else {
 *       // The PreTokenizers can still manipulate the normalized strings
 *       // so we do this anyway and will merge it back a NormalizedString
 *       preTok, err := t.doPreTokenize(sentence)
 *       if err != nil {
 *         log.Fatal(err)
 *       }
 *
 *       return preTok.Substring.Normalized
 *     }
 *   }
 * }
 *
 * //doPreTokenize does the PreTokenization logic, handling the case where there is no PreTokenizer set
 * func (t *Tokenizer) doPreTokenize(sentence string) (retVal PreTokenizedString, err error) {
 *   pretokenized := NewPreTokenizedString(sentence)
 *   if t.preTokenizer != nil {
 *     pretok, err := (*t.preTokenizer).PreTokenize(pretokenized)
 *   } else {
 *     return pretokenized
 *   }
 * }
 *  */
/*
 * func (t *Tokenizer) NumAddedTokens(isPair bool) int {
 *   return (*t.PostProcessor).AddedTokens(isPair)
 * }
 *
 * type splitRes struct {
 *   Content string
 *   Id      uint32
 *   Found   bool // whether split was found in AddedTokens/SpecialTokens
 * }
 *
 * // Encode encodes the given sentence
 * func (t *Tokenizer) Encode(input EncodeInput) Encoding {
 *
 *   var (
 *     sentence, pair         string
 *     encoding, pairEncoding Encoding
 *     isPair                 bool = false
 *   )
 *
 *   inputType := reflect.TypeOf(input)
 *
 *   switch inputType.Name() {
 *   case "Single":
 *     sentence = input.(Single).Sentence
 *     isPair = false
 *   case "Dual":
 *     sentence = input.(Dual).Sentence
 *     pair = input.(Dual).Pair
 *     isPair = true
 *   case "string":
 *     sentence = input.(string)
 *     isPair = false
 *   default:
 *     log.Fatalf("Unsupported type for input data type: %v. Input should have type of 'Single', 'Dual' or 'string' type.\n", inputType.Name())
 *   }
 *
 *   encoding = t.generateOutput(sentence, 0)
 *
 *   // 4. Post processing
 *   if t.PostProcessor != nil {
 *     if isPair {
 *       pairEncoding = t.generateOutput(pair, 1)
 *       return (*t.PostProcessor).Process(encoding, pairEncoding)
 *     } else {
 *       return (*t.PostProcessor).Process(encoding)
 *     }
 *   } else {
 *     if isPair {
 *       pairEncoding = t.generateOutput(pair, 1)
 *       encoding = t.postProcess(encoding, pairEncoding)
 *     } else {
 *       encoding = t.postProcess(encoding)
 *     }
 *
 *     // NOTE.Should we return pairEncoding as well?
 *     return encoding
 *   }
 * }
 *
 * func (t *Tokenizer) generateOutput(sentence string, typeId uint32) Encoding {
 *   // Split into as many sequences as needed to avoid splitting
 *   // on our added tokens
 *   var splits []splitRes
 *   var encodings []Encoding
 *
 *   splits = t.splitOnAddedTokens(sentence)
 *
 *   for _, s := range splits {
 *     // If this is one of our added tokens, return an encoding directly
 *     if s.Found {
 *       e := NewEncoding([]uint32{s.Id}, []uint32{typeId}, []string{s.Content}, []Offsets{{0, len(s.Content)}}, []uint32{0}, []uint32{1}, []Encoding{}) // TODO: add e.Words
 *
 *       encodings = append(encodings, e)
 *
 *     } else {
 *       // 1. Normalization
 *       var normalized normalizer.NormalizedString
 *       normalized = normalizer.NewNormalizedFrom(s.Content)
 *       if t.Normalizer != nil {
 *         nz := *t.Normalizer
 *         norm, err := nz.Normalize(normalized)
 *         if err != nil {
 *           log.Fatal(err)
 *         }
 *         normalized = norm
 *       }
 *
 *       // 2. Pre-tokenization
 *       var preTokenized *[]PreToken
 *       if t.PreTokenizer != nil {
 *         _, preTokenized = (*t.PreTokenizer).PreTokenize(&normalized)
 *       } else {
 *         str := normalized.GetNormalized()
 *         start := 0
 *         end := len(str)
 *         preToks := []PreToken{
 *           {
 *             Value: normalized.GetNormalized(),
 *             Offsets: Offsets{
 *               Start: start,
 *               End:   end,
 *             },
 *           },
 *         }
 *         preTokenized = &preToks
 *       }
 *
 *       fmt.Println("PreToken Offsets on Normalized string: ")
 *       for _, t := range *preTokenized {
 *         fmt.Printf("(%v %v)", t.Offsets.Start, t.Offsets.End)
 *       }
 *       fmt.Println()
 *
 *       fmt.Println("PreToken Offsets on Original string: ")
 *       for _, t := range *preTokenized {
 *         nRange := normalizer.NewRange(t.Offsets.Start, t.Offsets.End, normalizer.NormalizedTarget)
 *         oRange := normalized.ConvertOffset(nRange)
 *         fmt.Printf("(%v %v)", oRange.Start(), oRange.End())
 *       }
 *       fmt.Println()
 *
 *       // 3. Model
 *       output, err := (*t.Model).Tokenize(*preTokenized)
 *       if err != nil {
 *         log.Fatal(err)
 *       }
 *
 *       var en Encoding
 *       var offset int = 0
 *
 *       for _, t := range output {
 *         en.Ids = append(en.Ids, t.Id)
 *         en.Tokens = append(en.Tokens, t.Value)
 *
 *         start := t.Offsets.Start + offset
 *         end := t.Offsets.End + offset
 *         offset = end
 *
 *         // en.Offsets = append(en.Offsets, t.Offsets)
 *         en.Offsets = append(en.Offsets, Offsets{start, end})
 *         en.TypeIds = append(en.TypeIds, typeId)
 *         en.SpecialTokenMask = append(en.SpecialTokenMask, 0)
 *         en.AttentionMask = append(en.AttentionMask, 1)
 *       }
 *
 *       en.Overflowing = []Encoding{}
 *
 *       encodings = append(encodings, en)
 *     }
 *
 *   } // end loop over splits
 *
 *   if len(encodings) == 0 {
 *     return DefaultEncoding()
 *   }
 *
 *   // split off at position 1
 *   first := encodings[0]
 *   others := encodings[1:]
 *
 *   // Put others to overflowing of first
 *   for _, e := range others {
 *     first.MergeWith(e)
 *   }
 *
 *   return first
 * }
 *
 * // EncodeBatch encodes all sentences in concurrency
 * func (t *Tokenizer) EncodeBatch(inputs []EncodeInput) []Encoding {
 *   var encodings []Encoding
 *   var wg sync.WaitGroup
 *
 *   wg.Add(len(inputs))
 *
 *   // Encoding concurrently
 *   for i := 0; i < len(inputs); i++ {
 *     go func(i int) {
 *       defer wg.Done()
 *
 *       e := t.Encode(inputs[i])
 *       encodings = append(encodings, e)
 *
 *     }(i)
 *   }
 *
 *   wg.Wait()
 *
 *   // Do padding if included
 *   if t.Padding != nil {
 *     PadEncodings(encodings, *t.Padding)
 *   }
 *
 *   return encodings
 * }
 *
 * // Decode returns a corresponding string from an input id
 * func (t *Tokenizer) Decode(ids []uint32, skipSpecialTokens bool) string {
 *   var tokens []string
 *
 *   for _, id := range ids {
 *     // Look up at added tokens
 *     var token string
 *     tok, ok := t.AddedTokensR[id]
 *     if !ok {
 *       // Look up at model
 *       token, _ = t.IdToToken(id)
 *     }
 *
 *     token = tok.Content
 *
 *     _, ok = t.SpecialTokens[token]
 *
 *     if !skipSpecialTokens || !ok {
 *       tokens = append(tokens, token)
 *     }
 *   }
 *
 *   if t.Decoder != nil {
 *     return (*t.Decoder).Decode(tokens)
 *   }
 *
 *   return strings.Join(tokens, " ")
 * }
 *
 * // DecodeBatch decodes all sentences in concurrency
 * func (t *Tokenizer) DecodeBatch(sentences [][]uint32, skipSpecialTokens bool) []string {
 *   var decodings []string
 *   var wg sync.WaitGroup
 *
 *   wg.Add(len(sentences))
 *
 *   // Decoding concurrently
 *   for i := 0; i < len(sentences); i++ {
 *     go func(i int) {
 *       defer wg.Done()
 *
 *       s := t.Decode(sentences[i], skipSpecialTokens)
 *       decodings = append(decodings, s)
 *
 *     }(i)
 *   }
 *
 *   wg.Wait()
 *
 *   return decodings
 * }
 *
 * func (t *Tokenizer) splitOnAddedTokens(sentence string) []splitRes {
 *
 *   var splits []splitRes
 *   rs := []rune(sentence)
 *   var allSplits [][]int
 *
 *   // if there's no splitRe (regular epxression to split), do nothing
 *   if t.SplitRe == nil {
 *     splits = append(splits, splitRes{sentence, 0, false})
 *     return splits
 *   }
 *
 *   // matches contains slice of 2-element items (start and end byte position)
 *   // of the matched strings
 *   matches := t.SplitRe.FindAllStringIndex(sentence, -1)
 *
 *   // if no matches, just return the whole sentence
 *   if len(matches) == 0 {
 *     splits = append(splits, splitRes{sentence, 0, false})
 *     return splits
 *   }
 *
 *   for _, m := range matches {
 *     splits = append(splits, splitRes{
 *       Content: string(rs[m[0]:m[1]]),
 *       Id:      0,
 *     })
 *   }
 *
 *   // Collect also the splits in-between added tokens
 *   startOffset := 0
 *   for _, m := range matches {
 *     if startOffset < m[0] {
 *       allSplits = append(allSplits, []int{startOffset, m[0]})
 *     }
 *
 *     allSplits = append(allSplits, []int{m[0], m[1]})
 *     startOffset = m[1]
 *   }
 *
 *   // Check for the last piece
 *   fmt.Printf("Num of All Splits: %v\n", len(allSplits))
 *   fmt.Printf("All Splits: %v\n", allSplits)
 *   last := allSplits[len(allSplits)-1]
 *   if last[1] < len(sentence) {
 *     allSplits = append(allSplits, []int{last[1], len(sentence)})
 *   }
 *
 *   if len(allSplits) == 0 {
 *     splits = append(splits, splitRes{sentence, 0, false})
 *     return splits
 *   }
 *
 *   for _, m := range allSplits {
 *     s := string(rs[m[0]:m[1]])
 *     // Look up at special tokens
 *     id, ok := t.SpecialTokens[s]
 *     // not found. Look up at added tokens
 *     if !ok {
 *       // If not found, id will be 0 and ok = false
 *       id, ok = t.AddedTokens[AddedToken{
 *         Content:      s,
 *         IsSingleWord: false,
 *       }]
 *       if !ok {
 *         splits = append(splits, splitRes{
 *           Content: s,
 *           Id:      0,
 *           Found:   false,
 *         })
 *       }
 *     }
 *     splits = append(splits, splitRes{
 *       Content: s,
 *       Id:      id,
 *       Found:   true,
 *     })
 *   }
 *
 *   return splits
 *
 * }
 *
 * // Train trains a model and replaces the current model using a given trainer
 * // The tokenizer does the following steps
 * // 1. Concurrently, reads training data (text) from files, normalizes text using
 * // 		specified normalizer, and generates a slice of words and their frequency (count)
 * // 2. Train tokenizer model using specified tokenizer configuration on slice of word-count
 * //		generated from previous step to create `vocab` and `merges` data (files)
 * // 3. Update current tokenizer with newly generated model (`vocab` and `merges` data)
 * func (t *Tokenizer) Train(trainer Trainer, files []string) error {
 *   type Job struct {
 *     File     string
 *     Progress *progressbar.ProgressBar
 *   }
 *
 *   var jobs []Job
 *   wChan := make(chan map[string]uint32)
 *
 *   // channel to signal the main thread that all the words have been
 *   doneChan := make(chan (bool), 1)
 *   dict := make(map[string]uint32)
 *
 *   scanWG := new(sync.WaitGroup)
 *
 *   for _, f := range files {
 *     fsize, err := util.FileSize(f)
 *     if err != nil {
 *       log.Fatal(err)
 *     }
 *     bar := progressbar.New(int(fsize))
 *
 *     jobs = append(jobs, Job{f, bar})
 *   }
 *
 *   // Step 1. scan text files by chunks in goroutines. In each goroutine,
 *   // scan line by line, chop into tokens with (value, count) and
 *   // queue them up in a channel for next step.
 *   // We will set up a wait group to wait for all done.
 *   // For each file do:
 *   // 1. Create a goroutine to read file by chunks
 *   // 2. Read line by line
 *   // 3. Pre-tokenize line of text to tokens
 *   // 4. Process tokens into its value and count
 *   // 5. Send result to a channel for further processing.
 *   for i := 0; i < len(jobs); i++ {
 *     currentJob := i
 *
 *     file := jobs[currentJob].File
 *     // current is the counter for bytes of the file.
 *     var current int64 = 0
 *     var limit int64 = 100 * mb
 *
 *     fi, err := os.Stat(file)
 *     if err != nil {
 *       return err
 *     }
 *     fsize := float64(fi.Size())
 *
 *     chunkNum := int(math.Ceil(fsize / float64(limit)))
 *
 *     // Setup some workers to process
 *     for n := 1; n <= chunkNum; n++ {
 *       scanWG.Add(1)
 *
 *       go func(n int, file string) {
 *         // start reading file chunk by chunk
 *         current = t.processChunk(current, limit, file, wChan, trainer)
 *         fmt.Printf("File chunk %d has been completed\n", n)
 *         scanWG.Done()
 *       }(n, file)
 *     }
 *   }
 *
 *   // Read all incoming words from the channel and add to the dict
 *   go func() {
 *     fmt.Println("Start collecting words...")
 *     for words := range wChan {
 *       for w, c := range words {
 *         count, ok := dict[w]
 *         // word exists, sum up frequency
 *         if ok {
 *           dict[w] = count + c
 *         } else {
 *           // word not exist, let add it
 *           dict[w] = c
 *         }
 *       }
 *     }
 *
 *     // signal the main thread all done with this goroutine
 *     doneChan <- true
 *   }()
 *
 *   // wait for all goroutines to complete
 *   scanWG.Wait()
 *   close(wChan)
 *
 *   // Wait for dictionary to process all words then close
 *   <-doneChan
 *   close(doneChan)
 *
 *   fmt.Printf("Dictionary length: %v words\n", len(dict))
 *   // // Print out some samples
 *   // var count = 0
 *   // for k, _ := range dict {
 *   // if count <= 5 {
 *   // fmt.Println(k)
 *   // count++
 *   // }
 *   // }
 *
 *   // Training model
 *   fmt.Println("Start training...")
 *   model, specialTokens := trainer.Train(dict)
 *
 *   // Replace with trained model
 *   t.Model = &model
 *   t.AddSpecialTokens(specialTokens)
 *
 *   return nil
 * }
 *
 * // processChunk reads file chunk and processes it to word-count and sends off to channel
 * // offset: start bound
 * // limit: end bound
 * // filename: file path includes file name
 * // channel: channel to send proccessed words to.
 * // current: cummulative point where the file processing stops.
 * // trainer: Trainer to process tokens
 * func (t *Tokenizer) processChunk(offset int64, limit int64, filename string, channel chan (map[string]uint32), trainer Trainer) (current int64) {
 *   file, err := os.Open(filename)
 *   if err != nil {
 *     panic(err)
 *   }
 *   defer file.Close()
 *
 *   // move the pointer of the file to the start of designated chunk
 *   file.Seek(offset, 0) // 0 means relative to the origin of file
 *
 *   scanner := bufio.NewScanner(file)
 *   buf := make([]byte, 0, 1*gb) // initial buffer
 *   scanner.Buffer(buf, 2*gb)    // max buffer size = 2GB
 *
 *   var cummulativeSize int64
 *
 *   for scanner.Scan() {
 *     // Stop if read size has exceed the chunk size
 *     cummulativeSize += int64(len(scanner.Bytes()))
 *     if cummulativeSize > limit {
 *       break
 *     }
 *
 *     // line words
 *     lwords := make(map[string]uint32)
 *     var line string
 *     line = scanner.Text()
 *     // NOTE: io.scanner returns line w/o `\n`. We add it back manually.
 *     // line = fmt.Sprintf("%v\n", line)
 *
 *     normalized := t.normalize(line)
 *     // NOTE: if there are no preTokenizer, the default `preTokenize`
 *     // will return the whole line without modification. Hence,
 *     // token will be a line string. In that case, we may need to strip
 *     // white spaces in the next step.
 *     preTokenized := t.preTokenize(normalized.GetNormalized())
 *     var tokens []string
 *     for _, tok := range preTokenized {
 *       tokens = append(tokens, tok.Value)
 *     }
 *     // process tokens
 *     trainer.ProcessTokens(lwords, tokens)
 *     // send to channel for further process
 *     channel <- lwords
 *
 *   }
 *
 *   return cummulativeSize
 *
 * }
 *
 * func (t *Tokenizer) CTrain(trainer Trainer, files []string) error {
 *   type Job struct {
 *     File     string
 *     Progress *progressbar.ProgressBar
 *   }
 *
 *   var jobs []Job
 *
 *   for _, f := range files {
 *     fsize, err := util.FileSize(f)
 *     if err != nil {
 *       log.Fatal(err)
 *     }
 *     bar := progressbar.New(int(fsize))
 *
 *     jobs = append(jobs, Job{f, bar})
 *   }
 *
 *   // Doing jobs concurrently
 *
 *   g, ctx := errgroup.WithContext(context.Background())
 *   lnChan := make(chan map[string]uint32)
 *
 *   for i := 0; i < len(jobs); i++ {
 *     current := i
 *     g.Go(func() error {
 *       // Now, do the job
 *       file, err := os.Open(jobs[current].File)
 *       if err != nil {
 *         return err
 *       }
 *       defer file.Close()
 *
 *       var line string
 *       words := make(map[string]uint32)
 *
 *       scanner := bufio.NewScanner(file)
 *       for scanner.Scan() {
 *         line = scanner.Text()
 *         // io.scanner returns line w/o `\n`. We add it back manually.
 *         line = fmt.Sprintf("%v\n", line)
 *
 *         normalized := t.normalize(line)
 *         preTokenized := t.preTokenize(normalized.GetNormalized())
 *         var tokens []string
 *         for _, tok := range preTokenized {
 *           tokens = append(tokens, tok.Value)
 *         }
 *         trainer.ProcessTokens(words, tokens)
 *
 *         // Pass processed data to channel
 *         lnChan <- words
 *
 *         select {
 *         case lnChan <- words:
 *           // Keep going
 *         case <-ctx.Done():
 *           return ctx.Err()
 *         }
 *       }
 *
 *       if err := scanner.Err(); err != nil {
 *         return err
 *       }
 *
 *       return nil
 *
 *     })
 *   }
 *
 *   // Close out the channel when the first error occurs or
 *   // when processing is successful.
 *   go func() {
 *     g.Wait()
 *     close(lnChan)
 *   }()
 *
 *   err := g.Wait()
 *
 *   // as long as an error occurs, return it.
 *   if err != nil {
 *     return g.Wait()
 *   }
 *
 *   // Handle result coming from channel
 *   // words is a dictionary of words and their frequency
 *   words := make(map[string]uint32)
 *
 *   // calculate frequency and create a final map
 *   for result := range lnChan {
 *     fmt.Printf("Result: %v\n", result)
 *     for w, c := range result {
 *       count, ok := words[w]
 *       // word exists, sum up frequency
 *       if ok {
 *         words[w] = count + c
 *       }
 *       // word not exist, let add it
 *       words[w] = c
 *     }
 *   }
 *
 *   // Training model
 *   model, specialTokens := trainer.Train(words)
 *
 *   // Replace with trained model
 *   t.Model = &model
 *   t.AddSpecialTokens(specialTokens)
 *
 *   return nil
 *
 * }
 *
 * // PreTokenize processes logic, handling the case where there is no PreTokenizer set
 * func (t *Tokenizer) preTokenize(sentence string) []PreToken {
 *   // If there is no `PreTokenizer` setup, just return a slice
 *   // with one element of the whole string
 *   // TODO: should we split sentence into words?
 *   if t.PreTokenizer == nil {
 *     return []PreToken{
 *       {
 *         Value:   sentence,
 *         Offsets: Offsets{0, len(sentence)},
 *       },
 *     }
 *   }
 *
 *   normalized := normalizer.NewNormalizedFrom(sentence)
 *
 *   _, res := (*t.PreTokenizer).PreTokenize(&normalized)
 *   return *res
 * }
 *
 * // normalize normalizes using given normalizer
 * func (t *Tokenizer) normalize(sequence string) normalizer.NormalizedString {
 *   return normalizer.NewNormalizedFrom(sequence)
 * }
 *
 * // AddSpecialTokens registers give tokens as special tokens. This is especially useful
 * // for removing them while decoding.
 * func (t *Tokenizer) AddSpecialTokens(tokens []string) int {
 *   var addedTokens []AddedToken
 *   for _, tok := range tokens {
 *     addedTok := AddedTokenFrom(tok)
 *     addedTokens = append(addedTokens, addedTok)
 *
 *     // add to special tokens
 *     id, _ := t.TokenToId(tok)
 *     if id > 0 {
 *       t.SpecialTokens[tok] = id
 *     }
 *   }
 *
 *   added := t.AddTokens(addedTokens)
 *
 *   t.refreshAddedTokens()
 *
 *   return added
 *
 * }
 *
 * // AddTokens adds given tokens to added vocabulary
 * func (t *Tokenizer) AddTokens(tokens []AddedToken) int {
 *   var ignored = 0
 *   for _, tok := range tokens {
 *     _, ok := t.TokenToId(tok.Content)
 *     if len(tok.Content) == 0 || ok {
 *       ignored += 1
 *       continue
 *     }
 *
 *     newId := uint32((*t.Model).GetVocabSize()) + uint32(len(t.AddedTokens))
 *     id := t.AddedTokens[tok]
 *     // found
 *     if id > 0 {
 *       ignored += 1
 *     }
 *     // not found. Add it
 *     t.AddedTokens[tok] = newId
 *     // update the current revert map
 *     t.AddedTokensR[newId] = tok
 *
 *   }
 *
 *   t.refreshAddedTokens()
 *
 *   // Return the number of added tokens
 *   return len(tokens) - ignored
 * }
 *
 * // PostProcess processes the case where there is no PostProcessor set
 * func (t *Tokenizer) postProcess(encoding Encoding, pairEncodings ...Encoding) Encoding {
 *
 *   var (
 *     isPaired     bool = false
 *     pairEncoding Encoding
 *   )
 *   // 1. Truncate if needed
 *   if t.Trunc != nil {
 *     var nAddedTokens int
 *     if t.PostProcessor == nil {
 *       nAddedTokens = 0
 *     }
 *
 *     if pairEncodings != nil {
 *       isPaired = true
 *       pairEncoding = pairEncodings[0]
 *     }
 *     nAddedTokens = (*t.PostProcessor).AddedTokens(isPaired)
 *
 *     if nAddedTokens > 0 {
 *       params := t.Trunc
 *       params.MaxLength = t.Trunc.MaxLength - uint(nAddedTokens)
 *       TruncateEncodings(encoding, *params, pairEncoding)
 *     } else {
 *       TruncateEncodings(encoding, *t.Trunc, pairEncoding)
 *     }
 *   }
 *
 *   // 2. Post processing
 *   var finalEncoding Encoding
 *   if t.PostProcessor != nil {
 *     finalEncoding = (*t.PostProcessor).Process(encoding, pairEncoding)
 *   } else {
 *     if isPaired {
 *       finalEncoding = encoding
 *     }
 *
 *     encoding.MergeWith(pairEncoding)
 *     finalEncoding = encoding
 *   }
 *
 *   // 3. Padding if needed
 *   if t.Padding != nil {
 *     // We can only pad for a given size. If the Strategy is BatchLongest,
 *     // It will be done when we handle a batch
 *     var size uint
 *     if t.Padding.Strategy.Name == "Fixed" {
 *       size = t.Padding.Strategy.Value.(uint)
 *     } else {
 *       size = uint(len(finalEncoding.GetIds()))
 *     }
 *
 *     finalEncoding.Pad(size, t.Padding.PadId, t.Padding.PadTypeId, t.Padding.PadToken, t.Padding.Direction)
 *   }
 *
 *   return finalEncoding
 * }
 *
 * func (t *Tokenizer) refreshAddedTokens() {
 *   // We need to rebuild regexp here everytime
 *   // because the added tokens may have changed
 *   var specialTokens []AddedToken
 *   var newTokens []string
 *
 *   for k := range t.SpecialTokens {
 *     addedTok := AddedToken{
 *       Content:      k,
 *       IsSingleWord: true,
 *     }
 *     specialTokens = append(specialTokens, addedTok)
 *   }
 *
 *   var addedTokens []AddedToken
 *   for k := range t.AddedTokens {
 *     addedTokens = append(addedTokens, k)
 *   }
 *
 *   // merge with the one from special tokens
 *   addedTokens = append(addedTokens, specialTokens...)
 *
 *   for _, tok := range addedTokens {
 *     newTok := getPattern(tok)
 *     newTokens = append(newTokens, newTok)
 *   }
 *
 *   if len(newTokens) == 0 {
 *     t.SplitRe = nil
 *   }
 *
 *   re := strings.Join(newTokens, "|")
 *   t.SplitRe = regexp.MustCompile(re)
 * }
 *
 * func getPattern(tok AddedToken) string {
 *   var r string
 *   if tok.IsSingleWord {
 *     var (
 *       firstB string // first boundary
 *       lastB  string // last boundary
 *     )
 *     chars := strings.Split(tok.Content, "")
 *     firstChar := chars[0]
 *     lastChar := chars[len(chars)]
 *
 *     isWordChar := func(char string) bool {
 *       m, err := regexp.MatchString(`\w`, char)
 *       if err != nil {
 *         log.Fatal(err)
 *       }
 *       return m
 *     }
 *
 *     if isWordChar(firstChar) {
 *       firstB = fmt.Sprintf("%v", `\b`) // NOTE: back tick for raw string
 *     } else {
 *       firstB = ""
 *     }
 *
 *     if isWordChar(lastChar) {
 *       lastB = fmt.Sprintf("%v", `\b`)
 *     } else {
 *       lastB = ""
 *     }
 *
 *     // Escape all regular expression metacharacters
 *     // so the return is safe to use in a regular expression
 *     escapeTok := regexp.QuoteMeta(tok.Content)
 *     r = fmt.Sprintf("%v%v%v", firstB, escapeTok, lastB)
 *   } else {
 *     r = regexp.QuoteMeta(tok.Content)
 *   }
 *
 *   return r
 * } */