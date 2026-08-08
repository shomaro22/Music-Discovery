package usecase

type curatedCatalogProfile struct {
	Genre      string
	Subgenre   string
	Mood       string
	Element    string
	Atmosphere string
}

type CuratedCatalogSeed struct {
	Artist   string
	Track    string
	TagSlugs []string
}

var (
	catalogJPop = curatedCatalogProfile{
		Genre: "genre-pop", Subgenre: "subgenre-j-pop",
		Mood: "mood-uplifting", Element: "element-melody", Atmosphere: "atmosphere-bright",
	}
	catalogJRock = curatedCatalogProfile{
		Genre: "genre-rock", Subgenre: "subgenre-j-rock",
		Mood: "mood-uplifting", Element: "element-rhythm", Atmosphere: "atmosphere-refreshing",
	}
	catalogAlternative = curatedCatalogProfile{
		Genre: "genre-alternative", Subgenre: "subgenre-alternative-rock",
		Mood: "mood-extraordinary", Element: "element-worldview", Atmosphere: "atmosphere-deep",
	}
	catalogMelodicPunk = curatedCatalogProfile{
		Genre: "genre-rock", Subgenre: "subgenre-melodic-punk",
		Mood: "mood-uplifting", Element: "element-rhythm", Atmosphere: "atmosphere-refreshing",
	}
	catalogPunkRock = curatedCatalogProfile{
		Genre: "genre-rock", Subgenre: "subgenre-punk-rock",
		Mood: "mood-uplifting", Element: "element-rhythm", Atmosphere: "atmosphere-bright",
	}
	catalogPostHardcore = curatedCatalogProfile{
		Genre: "genre-alternative", Subgenre: "subgenre-post-hardcore",
		Mood: "mood-extraordinary", Element: "element-rhythm", Atmosphere: "atmosphere-deep",
	}
	catalogMetalcore = curatedCatalogProfile{
		Genre: "genre-metal", Subgenre: "subgenre-metalcore",
		Mood: "mood-extraordinary", Element: "element-rhythm", Atmosphere: "atmosphere-deep",
	}
	catalogMetal = curatedCatalogProfile{
		Genre: "genre-metal", Subgenre: "subgenre-heavy-metal",
		Mood: "mood-extraordinary", Element: "element-rhythm", Atmosphere: "atmosphere-deep",
	}
	catalogInternationalPop = curatedCatalogProfile{
		Genre: "genre-pop", Subgenre: "subgenre-international-pop",
		Mood: "mood-uplifting", Element: "element-vocal", Atmosphere: "atmosphere-bright",
	}
	catalogKPop = curatedCatalogProfile{
		Genre: "genre-k-pop", Subgenre: "subgenre-k-pop",
		Mood: "mood-uplifting", Element: "element-vocal", Atmosphere: "atmosphere-bright",
	}
)

func curatedCatalogArtist(artist string, profile curatedCatalogProfile, tracks ...string) []CuratedCatalogSeed {
	result := make([]CuratedCatalogSeed, 0, len(tracks))
	for _, track := range tracks {
		result = append(result, CuratedCatalogSeed{
			Artist: artist,
			Track:  track,
			TagSlugs: []string{
				profile.Genre,
				profile.Subgenre,
				profile.Mood,
				profile.Element,
				profile.Atmosphere,
			},
		})
	}
	return result
}

func joinCuratedCatalog(groups ...[]CuratedCatalogSeed) []CuratedCatalogSeed {
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	result := make([]CuratedCatalogSeed, 0, count)
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

// DefaultCuratedCatalogSeeds is a stable Japan-oriented foundation catalog.
// It favors well-known Japanese artists and songs, while deliberately giving
// melodic punk, punk, hardcore and metal enough depth for discovery.
// Spotify metadata is resolved at import time instead of being hard-coded.
func DefaultCuratedCatalogSeeds() []CuratedCatalogSeed {
	return joinCuratedCatalog(
		// Japanese pop and broadly popular Japanese music.
		curatedCatalogArtist("Mrs. GREEN APPLE", catalogJPop, "青と夏", "ライラック"),
		curatedCatalogArtist("back number", catalogJPop, "高嶺の花子さん", "水平線"),
		curatedCatalogArtist("米津玄師", catalogAlternative, "Lemon", "KICK BACK"),
		curatedCatalogArtist("Official髭男dism", catalogJPop, "Pretender", "Subtitle"),
		curatedCatalogArtist("Vaundy", catalogAlternative, "怪獣の花唄", "踊り子"),
		curatedCatalogArtist("YOASOBI", catalogJPop, "夜に駆ける", "アイドル"),
		curatedCatalogArtist("Ado", catalogAlternative, "うっせぇわ", "新時代"),
		curatedCatalogArtist("King Gnu", catalogAlternative, "白日", "SPECIALZ"),
		curatedCatalogArtist("あいみょん", catalogJPop, "マリーゴールド", "裸の心"),
		curatedCatalogArtist("藤井 風", catalogJPop, "きらり", "死ぬのがいいわ"),
		curatedCatalogArtist("優里", catalogJPop, "ドライフラワー", "ベテルギウス"),
		curatedCatalogArtist("ヨルシカ", catalogAlternative, "ただ君に晴れ", "花に亡霊"),
		curatedCatalogArtist("Creepy Nuts", catalogJPop, "Bling-Bang-Bang-Born", "のびしろ"),
		curatedCatalogArtist("BUMP OF CHICKEN", catalogJRock, "天体観測", "クロノスタシス"),
		curatedCatalogArtist("RADWIMPS", catalogJRock, "前前前世", "スパークル"),
		curatedCatalogArtist("スピッツ", catalogJPop, "チェリー", "美しい鰭"),
		curatedCatalogArtist("宇多田ヒカル", catalogJPop, "First Love", "One Last Kiss"),
		curatedCatalogArtist("LiSA", catalogJRock, "紅蓮華", "炎"),
		curatedCatalogArtist("Aimer", catalogJPop, "残響散歌", "カタオモイ"),
		curatedCatalogArtist("サカナクション", catalogAlternative, "新宝島", "忘れられないの"),

		// Japanese rock, melodic punk and punk.
		curatedCatalogArtist("ELLEGARDEN", catalogMelodicPunk, "Missing", "Space Sonic"),
		curatedCatalogArtist("MONOEYES", catalogMelodicPunk, "My Instant Song", "Run Run"),
		curatedCatalogArtist("the HIATUS", catalogAlternative, "Insomnia", "Horse Riding"),
		curatedCatalogArtist("10-FEET", catalogMelodicPunk, "第ゼロ感", "RIVER"),
		curatedCatalogArtist("Hi-STANDARD", catalogMelodicPunk, "My First Kiss", "Stay Gold"),
		curatedCatalogArtist("Ken Yokoyama", catalogMelodicPunk, "Believer", "Maybe Maybe"),
		curatedCatalogArtist("dustbox", catalogMelodicPunk, "Jupiter", "Tomorrow"),
		curatedCatalogArtist("locofrank", catalogMelodicPunk, "START", "reason"),
		curatedCatalogArtist("Northern19", catalogMelodicPunk, "MESSAGE", "STAY YOUTH FOREVER"),
		curatedCatalogArtist("SHANK", catalogMelodicPunk, "Take Me Back", "Good Night Darling"),
		curatedCatalogArtist("HEY-SMITH", catalogPunkRock, "Be The One", "Fellowship Anthem"),
		curatedCatalogArtist("04 Limited Sazabys", catalogMelodicPunk, "swim", "fiction"),
		curatedCatalogArtist("WANIMA", catalogMelodicPunk, "ともに", "シグナル"),
		curatedCatalogArtist("TOTALFAT", catalogPunkRock, "Place to Try", "PARTY PARTY"),
		curatedCatalogArtist("GOOD4NOTHING", catalogMelodicPunk, "It's My Paradise", "Cause You're Alive"),
		curatedCatalogArtist("OVER ARM THROW", catalogMelodicPunk, "ZINNIA", "Thanks"),
		curatedCatalogArtist("ENTH", catalogPunkRock, "Get Started Together", "BONG! CAFE' AU LAIT! ACHE!"),
		curatedCatalogArtist("ROTTENGRAFFTY", catalogJRock, "金色グラフティー", "D.A.N.C.E."),
		curatedCatalogArtist("SiM", catalogPostHardcore, "The Rumbling", "KiLLiNG ME"),
		curatedCatalogArtist("coldrain", catalogPostHardcore, "MAYDAY", "ENVY"),

		// Japanese and international metal / hardcore.
		curatedCatalogArtist("Crossfaith", catalogMetalcore, "Monolith", "Jägerbomb"),
		curatedCatalogArtist("Crystal Lake", catalogMetalcore, "Apollo", "Disobey"),
		curatedCatalogArtist("FACT", catalogPostHardcore, "a fact of life", "slip of the lip"),
		curatedCatalogArtist("MAXIMUM THE HORMONE", catalogMetal, "What's up, people?!", "恋のメガラバ"),
		curatedCatalogArtist("BABYMETAL", catalogMetal, "ギミチョコ！！", "メギツネ"),
		curatedCatalogArtist("SABLE HILLS", catalogMetalcore, "Crisis", "Odyssey"),
		curatedCatalogArtist("Bring Me The Horizon", catalogMetalcore, "Can You Feel My Heart", "Throne"),
		curatedCatalogArtist("Architects", catalogMetalcore, "Animals", "Doomsday"),
		curatedCatalogArtist("Parkway Drive", catalogMetalcore, "Wild Eyes", "Vice Grip"),
		curatedCatalogArtist("Killswitch Engage", catalogMetalcore, "My Curse", "The End of Heartache"),

		// International artists popular in Japan, with a rock-heavy balance.
		curatedCatalogArtist("Linkin Park", catalogAlternative, "Numb", "In the End"),
		curatedCatalogArtist("Green Day", catalogPunkRock, "Basket Case", "American Idiot"),
		curatedCatalogArtist("Sum 41", catalogPunkRock, "Fat Lip", "Still Waiting"),
		curatedCatalogArtist("blink-182", catalogPunkRock, "All the Small Things", "What's My Age Again?"),
		curatedCatalogArtist("The Offspring", catalogPunkRock, "The Kids Aren't Alright", "You're Gonna Go Far, Kid"),
		curatedCatalogArtist("My Chemical Romance", catalogAlternative, "Welcome to the Black Parade", "Teenagers"),
		curatedCatalogArtist("Taylor Swift", catalogInternationalPop, "Cruel Summer", "Anti-Hero"),
		curatedCatalogArtist("Bruno Mars", catalogInternationalPop, "Just the Way You Are", "Locked out of Heaven"),
		curatedCatalogArtist("BTS", catalogKPop, "Dynamite", "Butter"),
		curatedCatalogArtist("TWICE", catalogKPop, "TT", "What Is Love?"),
	)
}
