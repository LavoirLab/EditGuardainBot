package utils

import (
	"sync"
	"unicode"
	"unicode/utf8"
)

type trieNode struct {
	children map[rune]*trieNode
	isEnd    bool
}

var (
	abuseTrie = &trieNode{children: make(map[rune]*trieNode)}
	trieOnce  sync.Once
)

func populateTrie() {
	for word := range abusiveWords {
		node := abuseTrie
		for _, r := range word {
			r = unicode.ToLower(r)
			if node.children == nil {
				node.children = make(map[rune]*trieNode)
			}
			child, ok := node.children[r]
			if !ok {
				child = &trieNode{}
				node.children[r] = child
			}
			node = child
		}
		node.isEnd = true
	}
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func ContainsAbusiveWord(sentence string) bool {
	trieOnce.Do(populateTrie)

	var lastRune rune = -1
	for i, r := range sentence {
		// Only start matching at the beginning or after a non-word character
		if lastRune == -1 || !isWordChar(lastRune) {
			node := abuseTrie
			tempIdx := i
			var lastMatchRune rune

			// Look ahead in the Trie
			for _, rr := range sentence[i:] {
				lowerRR := unicode.ToLower(rr)
				if node.children == nil {
					break
				}
				child, ok := node.children[lowerRR]
				if !ok {
					break
				}
				node = child
				rrSize := utf8.RuneLen(rr)
				tempIdx += rrSize
				lastMatchRune = rr

				if node.isEnd {
					// Check word boundary at the end
					if isWordChar(lastMatchRune) {
						if tempIdx < len(sentence) {
							nextR, _ := utf8.DecodeRuneInString(
								sentence[tempIdx:],
							)
							if isWordChar(nextR) {
								continue // Potential substring, keep looking for longer match
							}
						}
					}
					return true
				}
			}
		}
		lastRune = r
	}
	return false
}

var abusiveWords = map[string]struct{}{
	"18+": {}, "69": {}, "aad": {}, "aalsi": {}, "aand": {}, "adult": {}, "all groups available dm me": {},
	"all type video available jisko chahiye just aao fast": {}, "anal": {},
	"any couple want some\xa0 gang bang cuckolding and pregnancy": {}, "any cut guy here": {}, "any girl for call": {},
	"any girl for call then dm me": {}, "any girl from delhi in this group": {},
	"any girl from lucknow in this group": {}, "any girl from mumbai in this group": {},
	"any girl from up in this group": {}, "any girl want to roleplay dm only genuine girls": {},
	"any one women for fun chat": {}, "any sugar mommy here or 20+ female": {},
	"anyone want to enjoy mutual then come": {}, "arsehole": {}, "asshat": {}, "asshole": {}, "audio call": {},
	"aunty ki chut": {}, "aunty lover dm": {}, "available now": {}, "b***chod": {}, "b**chod": {}, "b.c.": {},
	"b.s.d.k": {}, "babbe": {}, "babbey": {}, "backside show": {}, "baddimag": {}, "badmash": {}, "badtameez": {},
	"bahenchod": {}, "bakchod": {}, "bakchodd": {}, "bakchodi": {}, "bakwas": {}, "bastard": {}, "bathroom sex": {},
	"bc": {}, "bdsm": {}, "behenchod": {}, "besharam": {}, "betichod": {}, "bevakoof": {}, "bevda": {}, "bevdey": {},
	"bevkoof": {}, "bewda": {}, "bewday": {}, "bhabhi chudai": {}, "bhabhi ki chut": {}, "bhabhi lover dm": {},
	"bhabhi nude": {}, "bhabhi nudes": {}, "bhabhi sex": {}, "bhadua": {}, "bhaduaa": {}, "bhadva": {}, "bhadvaa": {},
	"bhadwa": {}, "bhadwaa": {}, "bhagoda": {}, "bhains": {}, "bhains ki aankh": {}, "bhenchod": {}, "bhenchodd": {},
	"bhikhari": {}, "bhonsdike": {}, "bhosada": {}, "bhosadchod": {}, "bhosadchodal": {}, "bhosadike": {}, "bhosda": {},
	"bhosdaa": {}, "bhosdi": {}, "bhosdike": {}, "bhosdiki": {}, "bhosdiwala": {}, "bhosdiwale": {}, "big ass": {},
	"big black cock": {}, "big boobs": {}, "big cock": {}, "bikau": {}, "bikini photos": {}, "bio link join": {},
	"bio links join all girls": {}, "bitch": {}, "bkl": {}, "bloody": {}, "blowjob": {}, "bobs": {},
	"body to body massage": {}, "bollocks": {}, "bollywood hot": {}, "bollywood sex": {}, "boobs": {}, "boobs press": {},
	"boops": {}, "boor": {}, "boy to boy dm me": {}, "boy's come inbox": {}, "bsdk": {}, "bube": {}, "bubey": {},
	"budbak": {}, "bugger": {}, "bullshit": {}, "bur": {}, "burr": {}, "buur": {}, "buurr": {}, "call girl": {},
	"can't send here anybodies dm me i will show you it's free": {}, "casting couch": {}, "ch**d": {}, "chaalu": {},
	"chalu": {}, "chapri": {}, "charsi": {}, "chhinar": {}, "chhod": {}, "chichora": {}, "chicken shit": {},
	"child porn": {}, "chinal ka bacha": {}, "chindi": {}, "ching chong": {}, "chirkut": {}, "chod": {}, "chodd": {},
	"chodu": {}, "choli romance": {}, "chooche": {}, "choochi": {}, "choor": {}, "choot": {}, "chuchi": {}, "chudai": {},
	"chudai story": {}, "chudne": {}, "chudney": {}, "chudwa": {}, "chudwaa": {}, "chudwaane": {}, "chudwane": {},
	"chut": {}, "chut ke": {}, "chut ke bache": {}, "chut ke laude": {}, "chutad": {}, "chute": {}, "chutia": {},
	"chutiya": {}, "chutiyapa": {}, "chutiye": {}, "chuttad": {}, "cleavage show": {}, "clusterfuck": {},
	"cm show available dm": {}, "cock": {}, "cocksucker": {}, "come on inbox": {}, "condom": {}, "coonass": {},
	"cornhole": {}, "cox–zucker machine": {}, "cracker": {}, "crap": {}, "cum": {}, "cunt": {},
	"dad body guy with a plus size d interested gorgeous ladies please dm": {}, "dakho biioo aa jao obbii..oo aa jao": {},
	"dalaal": {}, "dalal": {}, "dalle": {}, "dalley": {}, "deepthroat": {}, "demo 1 pic free": {}, "desi call girl": {},
	"desi nudes": {}, "desi randi": {}, "desi romance": {}, "desi sex": {}, "desi sex video": {}, "desi thighs": {},
	"dhakkan": {}, "dhongi": {}, "dhurt": {}, "dick": {}, "dildo": {}, "dirty talk": {}, "dirty video": {},
	"dirtyy talkss with giffs": {}, "dm karo baby": {}, "dm me for service": {}, "dumbass": {}, "enshittification": {},
	"erotic": {}, "erotic massage": {}, "escort": {}, "exposed": {}, "faggot": {}, "faltu": {}, "farzi": {}, "fattu": {},
	"fattuu": {}, "feck": {}, "forced porn": {}, "fraud": {}, "free earning krne k liye msg kro": {}, "free now": {},
	"fuck": {}, "fuck her right in the pussy": {}, "fuck joe biden": {}, "fuck me": {}, "fuck you": {},
	"fuck, marry, kill": {}, "fuckery": {}, "full open service": {}, "futa": {}, "g***i": {}, "gaand": {}, "gadha": {},
	"gadhalund": {}, "gadhe": {}, "gand": {}, "gand ka keeda": {}, "gand mara": {}, "ganda": {}, "gandfat": {},
	"gandfut": {}, "gandiya": {}, "gandiye": {}, "gandu": {}, "gappi": {}, "gay dm me": {}, "gay sex": {},
	"gb road ke paidaish": {}, "gb road pe teri ma show karti": {}, "genuine boys dm": {}, "ghamandi": {}, "ghatiya": {},
	"gils cann dmmm me": {}, "girls chahiye dm me": {}, "goo": {}, "gote": {}, "gotey": {}, "gotte": {},
	"grab 'em by the pussy": {}, "gu": {}, "h koii mood m": {}, "hag": {}, "haggu": {}, "hagne": {}, "hagney": {},
	"handjob": {}, "haraamjaada": {}, "haraamjaade": {}, "haraamkhor": {}, "haraamzaade": {}, "haraamzyaada": {},
	"haram ki aulaad": {}, "harami": {}, "haramjada": {}, "haramkhor": {}, "haramzaada": {}, "haramzada": {},
	"haramzyada": {}, "hard cock": {}, "healslut": {}, "hello koi ladki chudegi": {}, "hentai": {},
	"here's 8 inches for you gujju boy": {}, "hidden cam": {}, "hijab porn": {}, "hijda": {}, "hijra sex": {},
	"hila rha hu koi dekhogi msg kro": {}, "honeymoon night": {}, "honeymoon sex": {}, "hot cam": {}, "hot chat": {},
	"hot desi bhabhi": {}, "hot girl": {}, "hot indian actress": {}, "hot indian model": {}, "hot navel": {},
	"hot romance": {}, "hot saree": {}, "hot video": {}, "hotel sex": {}, "hotwife": {},
	"hyy everyone i need cute girl friend ant girl here": {}, "i am ready message me": {},
	"i m uncle genuine girl cone": {}, "i need girlfriend": {}, "i need girlfriend for sex chat": {},
	"i want service message me": {}, "i'm a pregnancy provider dm me": {}, "i'm available came boy": {},
	"i'm full nude now": {}, "i'm genuine experience m bull from": {}, "i'm well educated bull looking good": {},
	"if you see kay": {}, "inbox karo baby": {}, "incest": {}, "item girl": {}, "jesus fucking christ": {}, "jhaat": {},
	"jhaatu": {}, "jhantu": {}, "jhat": {}, "jhatu": {}, "jism": {}, "jo bhabhi satisfied nhi hui hai yha": {},
	"join bio link": {}, "join my chatting group": {}, "join naughty group": {}, "join rape video channel": {},
	"join viral video channel": {}, "kachra": {}, "kamina": {}, "kaminey": {}, "kamzarf": {}, "kanjoos": {}, "kike": {},
	"kinky": {}, "kisi ladki ka ko video chahiye ho to message kro": {},
	"kisi larki ko apna paani nikalwana hai toh ajaie": {}, "koi achhi ldki hai kya jo bat krna chahegi": {},
	"koi bhabhi h jo 8 inch tool here any house wife jo pyasi ho": {},
	"koi boy mera friend  ya bestu ya boy friend banega":          {},
	"koi ladki h jo baat karegi   paani  na nikal du to bolnaaa":  {}, "kutia": {}, "kutiya": {}, "kutta": {},
	"kutta kamina": {}, "kutte": {}, "kuttey": {}, "kutti": {}, "kuttiya": {}, "lafanga": {}, "landi": {}, "landy": {},
	"lauda": {}, "laude": {}, "laudey": {}, "launda": {}, "laundey": {}, "laundi": {}, "laundiya": {}, "laura": {},
	"lesbian": {}, "lick pussy": {}, "licking": {}, "ling": {}, "lingam massage": {}, "lingerie": {}, "live sex": {},
	"loda": {}, "lode": {}, "lodu": {}, "loli": {}, "lora": {}, "lounde": {}, "loundi": {}, "loundiya": {},
	"love making": {}, "low rate videos available please dm": {}, "lukkha": {}, "lulli": {}, "lund": {}, "lund mar": {},
	"lundoo": {}, "lust": {}, "lutera": {}, "m**ch*d": {}, "m.c.": {}, "maal": {}, "madarchod": {}, "madarchodd": {},
	"madarchood": {}, "madarchoot": {}, "madarchut": {}, "madharchod": {}, "madharchodh apni bahan mat chuda": {},
	"madharchodh apni ma mat chuda": {}, "madhrachod": {}, "mamme": {}, "mammey": {},
	"married ladies can you send me a message to chat": {}, "massage sex": {}, "masturbation": {}, "mc": {}, "mcchod": {},
	"mere pass aayhe mera hatiyaar bhot aacha h aao dm kro bhabhiyo": {}, "message me fast baby": {},
	"message me personally": {}, "milf": {}, "mkc": {}, "mkl": {}, "moaning": {}, "mom lover dm": {},
	"mom sister sleeping group is available at low price": {}, "moot": {}, "mooth": {}, "mootne": {}, "mother chor": {},
	"motherfucker": {}, "msg me": {}, "msg me boys": {}, "msg me hot skoooll boy here": {}, "mut": {}, "muth": {},
	"muthi": {}, "mutne": {}, "my milky girlfriend pics": {}, "naked": {}, "naked selfie": {}, "nakli": {}, "nalayak": {},
	"nalla": {}, "nangi": {}, "nangi ladki": {}, "naughty": {}, "naughty chat": {}, "need boyfriend": {}, "nigga": {},
	"nigger": {}, "nikkamma": {}, "nipple": {}, "nude": {}, "nude aunty": {}, "nudes viral": {}, "nunni": {}, "nunnu": {},
	"only for today offer": {}, "onlyfans": {}, "paaji": {}, "pagal": {}, "pagal kutta": {}, "paid service available": {},
	"paid show": {}, "paid videos available anyone need please dm": {}, "pajeet": {}, "paji": {}, "paki": {},
	"panauti": {}, "patreon nudes": {}, "pedo": {}, "penis": {}, "pesaab": {}, "pesab": {}, "peshaab": {}, "peshab": {},
	"pilla": {}, "pillay": {}, "pille": {}, "pilley": {}, "pisaab": {}, "pisab": {}, "pkmkb": {},
	"plzzz commmmm tooooo myyy biiioooo gyyzzz": {}, "poof": {}, "poofter": {}, "porkistan": {}, "porn": {}, "prick": {},
	"private group chalogi baby": {}, "private parts": {}, "pussy": {}, "raand": {}, "rand": {}, "randa": {}, "randi": {},
	"randi ka bacha": {}, "randi ke pille tore ma ko chodho": {}, "randi khana": {}, "randi ki aulaad": {}, "randy": {},
	"rape porn": {}, "rape video": {}, "ratfucking": {}, "real meet": {}, "really meet only girls": {}, "retard": {},
	"ruk madhar chodh": {}, "russian warship, go fuck yourself": {}, "saala": {}, "sadela": {}, "sale": {}, "sandass": {},
	"saree romance": {}, "saree sex": {}, "schoolgirl porn": {}, "screwing": {},
	"se x x x gif  chat ttt kregi koi dmm kooo": {}, "see my story": {}, "send me a message to get added now": {},
	"sensual": {}, "service available": {}, "serving cunt": {}, "sex": {}, "sex cam": {}, "sex chat": {},
	"sex chat karogi": {}, "sex chat karogi baby": {}, "sex position": {}, "sex story": {}, "sex tape": {},
	"sex video": {}, "sex worker": {}, "sexy": {}, "sexy aunty": {}, "sexy aunty video": {}, "sexy dance": {},
	"sexy night": {}, "sexy photos": {}, "sexy saree": {}, "sexy time": {}, "shit": {}, "shit happens": {},
	"shithouse": {}, "shitposting": {}, "shitter": {}, "shut the fuck up": {}, "shut the hell up": {},
	"skoool boy here": {}, "sleeping pills grp available": {}, "slut": {}, "son of a bitch": {}, "spic": {}, "squirt": {},
	"stepbro": {}, "stepmom": {}, "stepsis": {}, "strip club": {}, "strip tease": {}, "suar": {}, "suhagraat sex": {},
	"sust": {}, "suvar": {}, "suvar ka baccha": {}, "swinger": {}, "taking the piss": {}, "tantra sex": {}, "tatte": {},
	"tatti": {}, "teen sex": {}, "tera bap gay hain sayad": {}, "tera dimaag kharab hai": {},
	"tera hot video viral karunga": {}, "tera porn banauga": {}, "teri bahan ki boor ungali karu": {},
	"teri bahan ki chut fadhu": {}, "teri bahan ki gand mai ungali karu": {}, "teri bahan meri rakhail": {},
	"teri band baja dunga": {}, "teri behan ki chut": {}, "teri gaand maar dunga": {}, "teri ma ka boor": {},
	"teri ma ka boor fadhu": {}, "teri ma ka kya ghufa hai kya": {}, "teri ma ki chut mai ungali kardu": {},
	"teri ma ki surang dekhi maine": {}, "teri ma ko patak ke chodh dunga": {}, "teri ma meri rakhail": {},
	"teri maa ka bhosda": {}, "teri maa ka doodh laa laa jaldii": {}, "teri maa ki chut": {},
	"teri mmmaaaaa kaaaa bhooosssssddddddaaaaaaa": {}, "teri taang tod dunga": {}, "teri waat laga dunga": {},
	"tharki": {}, "tight pussy": {}, "tits": {}, "tmc": {}, "tu apni ma chudwa le paise de dunga": {},
	"tum johny sin ke lannddd ka bal ho": {}, "twat": {}, "ulloo": {}, "ullu ka pattha": {}, "uncensored": {},
	"uncle here only female gil come": {}, "unclefucker": {}, "underage porn": {}, "unsatisfy girl": {},
	"use of nigger in proper names": {}, "vagina": {}, "vc aa teri maa ki chuut": {}, "video call available": {},
	"viral video": {}, "wanker": {}, "want service and exclusive video so dm me fast": {}, "wet pussy": {}, "whore": {},
	"wife swapping": {}, "with face fingaring no xtra charges": {}, "xxx": {}, "yaha koi bhabhi ya sister hai kya": {},
	"yt premium available": {}, "आँड": {}, "आंड": {}, "आंड़": {}, "कुतिया": {}, "कुत्ता": {}, "कुत्ती": {}, "कुत्ते": {},
	"गंडफट": {}, "गंडिया": {}, "गंडिये": {}, "गधा": {}, "गधालंड": {}, "गधे": {}, "गांड": {}, "गांडू": {}, "गू": {},
	"गोटे": {}, "चरसी": {}, "चुची": {}, "चुटिया": {}, "चुत्तड़": {}, "चुदने": {}, "चुदवा": {}, "चुदवाने": {}, "चूची": {},
	"चूचे": {}, "चूत": {}, "चूतिया": {}, "चूतिये": {}, "चूत्तड़": {}, "चोद": {}, "झाट": {}, "झाटू": {}, "दलले": {},
	"दलाल": {}, "नुननी": {}, "नुननु": {}, "पिल्ला": {}, "पिल्ले": {}, "पिसाब": {}, "पेशाब": {}, "पेसाब": {}, "फट्टू": {},
	"बकचोद": {}, "बकचोदी": {}, "बब्बे": {}, "बहनचोद": {}, "बुर": {}, "बूबे": {}, "बेवकूफ": {}, "बेवड़ा": {}, "बेवड़े": {},
	"बेहेनचोद": {}, "भड़वा": {}, "भड़ुआ": {}, "भेनचोद": {}, "भोसड़ा": {}, "भोसड़ाचोद": {}, "भोसड़ाचोदल": {},
	"भोसड़ीकी": {}, "भोसड़ीके": {}, "भोसड़ीवाला": {}, "भोसड़ीवाले": {}, "भोसदचोद": {}, "भोसरचोदल": {}, "मम्मे": {},
	"मादरचुत": {}, "मादरचूत": {}, "मादरचोद": {}, "मुठ": {}, "मुत": {}, "मुतने": {}, "मूठ": {}, "मूत": {}, "मूतने": {},
	"रंडी": {}, "रांड": {}, "लंड": {}, "लिंग": {}, "लुल्ली": {}, "लेंडी": {}, "लोड़ा": {}, "लोड़े": {}, "लोडा": {},
	"लोडे": {}, "लौंडा": {}, "लौंडिया": {}, "लौंडी": {}, "लौंडे": {}, "लौड़ा": {}, "लौड़े": {}, "लौडा": {}, "सुअर": {},
	"सूअर": {}, "हग": {}, "हगने": {}, "हग्गू": {}, "हरामखोर": {}, "हरामज़ादा": {}, "हरामज़ादे": {}, "हरामजादा": {},
	"हरामजादे": {}, "हरामी": {},
}
