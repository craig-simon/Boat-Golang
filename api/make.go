package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Make is a manufacturer of boats
type Make struct {
	ID         int          `json:",omitempty"`
	MIC        string       `json:",omitempty"`
	Name       string       `json:",omitempty"`
	Rank       int          `json:",omitempty"`
	YearRanges [][]int      `json:",omitempty"`
	Details    []MakeDetail `json:",omitempty"`
}

// MakeDetail is a unique kind of boat, including Year, Make, Model, Length, etc.
type MakeDetail struct {
	ID            int                       `json:",omitempty"`
	Year          int                       `json:",omitempty"`
	Series        string                    `json:",omitempty"`
	Model         string                    `json:",omitempty"`
	Locomotion    string                    `json:",omitempty" enum:"Power, Sail"`
	Type          string                    `json:",omitempty" enum:"Air Boats, Bay Launch, Catamaran, Houseboats, Hovercraft Boats, Inboard Boats, Inflatable Boats, Jet Drive Boats, L-Drive Boats, Monohull Sailboats, Outboard Boats, Pontoon Boats, Power Cat, Rowboats Driftboats Etc, Sea Drive Boats, Stern Drive Power Boat, Surface Drive Boats, Trimaran Boats, Utility/Jon, VDR"`
	Length        float32                   `json:",omitempty"`
	Beam          float32                   `json:",omitempty"`
	Weight        float32                   `json:",omitempty"`
	HullMaterials []string                  `json:",omitempty" enum:"Aluminum, Carbon Fiber, Composite, Epoxy, Fiberglass, Foam, Graphite Composite, Hypalon, Kevlar, Neoprene, Nylon, Plastic, Plywood, Polyester, Polyethylene, Polypropylene, Polyurethene, Resin Transfer Molding, Resitex, Rubber, Sheet Molded Compound, Steel, Strongnan, Wood"`
	EngineCount   int                       `json:",omitempty"`
	EnginePower   float32                   `json:",omitempty"`
	FuelType      string                    `json:",omitempty" enum:"Unknown, Gas, Diesel, Electric, Other"`
	Options       map[string]map[string]int `json:",omitempty"`
}

// NADA Marine Consumer database, converted to our structure and cached in memory
type NADA struct {
	Makes            map[int]*Make                             // i.e., nadaCache.Makes[11874] = Make{Name: "Twin Vee Powercats Inc", ...}
	MakeNameKeys     map[string][]int                          // i.e., nadaCache.MakeNameKeys["ADRENALINE"] = []int{10015, 10016}
	YearPowerOptions map[int]map[int]map[string]map[string]int // i.e., nadaCache.YearPowerOptions[2009][true]["Canvas"]["BIMINI TOP"] = 16000000001
}

var nadaCache = NADA{
	Makes:            map[int]*Make{},
	MakeNameKeys:     map[string][]int{},
	YearPowerOptions: map[int]map[int]map[string]map[string]int{},
}

var makes = map[int]*Make{
	// these will be filled in by addNADAMakes
	// 10000: Make{Name: "A & L Fiberglass", Rank: 2, YearRanges: [][]int{{2001, 2016}}},
	// 10001: Make{Name: "A & M Manufacturing Inc", Rank: 2, YearRanges: [][]int{{2002, 2014}}},
	// 10217: Make{Name: "Blackfin", Rank: 2, YearRanges: [][]int{{1984, 2001}, {2004, 2004}, {2018, 2018}}},
	// 10478: Make{Name: "Crusader Boats", Rank: 2, YearRanges: [][]int{{1984, 1986}, {1992, 1992}, {1994, 1997}, {2005, 2009}}},
}

func init() {
	addEnumsFor(MakeDetail{})
	apiHandlers["GetMakes"] = GetMakes
}

func startMake() {
	getNADACache()
	for makeID, make := range nadaCache.Makes {
		makes[makeID] = &Make{Name: make.Name, Rank: make.Rank, YearRanges: make.YearRanges}
	}
	// harvestBS(&request{Session: &Session{User: &User{ID: 1234567890123456}}, QA: true}, nil)
}

func getNADACache() {
	path := "nada/cache.json"
	if nadaCacheJSON, err := ioutil.ReadFile(path); err == nil {
		if err = json.Unmarshal(nadaCacheJSON, &nadaCache); err == nil {
			return
		}
	}
	// make cache.json the first time from csv files exported from NADA's Access database, MarineConsumer_EngAdjV###.mdb
	rankCompanies := map[int]map[string]string{}
	for rank := 1; rank <= 2; rank++ {
		path := "nada/Rank" + strconv.Itoa(rank) + ".txt"
		content, err := ioutil.ReadFile(path)
		if err != nil {
			panic(errors.New("Can't read " + path))
		}
		rankCompanies[rank] = map[string]string{}
		for _, companyName := range strings.Split(string(content), "\r\n") {
			rankCompanies[rank][strings.ToUpper(companyName)] = companyName
		}
	}
	validYearList := regexp.MustCompile(`^(\d{4}(,\d{4})*)?$`)
	_, validDetailTypes := Enums(MakeDetail{}, "Type")
	_, validHullMaterials := Enums(MakeDetail{}, "HullMaterials")
	validEng := regexp.MustCompile(`^(\d+)<br>(\d+(?:\.\d+)?) HP <br>(\w+)$`)
	_, validFuelTypes := Enums(MakeDetail{}, "FuelType")
	validOptionCat := regexp.MustCompile(`^(POWER BOAT|SAILBOAT):(.*)$`)
	nadaCompanies := readNADAFile("Companies", []string{"CompanyNum", "Company", "NotesCompany", "ModelYears", "Version"})
	makeIDs := map[int]bool{}
	for row := 0; row < nadaCompanies.Len; row++ {
		// CompanyNum should be unique integer
		companyNum := nadaCompanies.get(row, "CompanyNum")
		makeID, err := strconv.Atoi(companyNum)
		if err != nil {
			panic(errors.New("NADA Companies has CompanyNum \"" + companyNum + "\""))
		}
		if _, ok := makeIDs[makeID]; ok {
			panic(errors.New("NADA Companies has duplicate CompanyNum \"" + companyNum + "\""))
		}
		makeIDs[makeID] = true
		// ModelYears should be comma-separated list of 4-digit years
		modelYears := nadaCompanies.get(row, "ModelYears")
		if !validYearList.MatchString(modelYears) {
			panic(errors.New("NADA Companies has ModelYears \"" + modelYears + "\""))
		}
		// add Make to cache
		name := nadaCompanies.get(row, "Company")
		rank := 1
		for rank <= 2 {
			if titleCase, ok := rankCompanies[rank][name]; ok {
				name = titleCase
				break
			}
			rank++
		}
		if rank == 3 {
			name = strings.Title(strings.ToLower(name))
		}
		nadaCache.Makes[makeID] = &Make{
			ID:         makeID,
			Name:       name,
			Rank:       rank,
			YearRanges: yearListToYearRanges(modelYears),
			Details:    []MakeDetail{},
		}
		makeNameKey := getMakeNameKey(name)
		if makeIDList, ok := nadaCache.MakeNameKeys[makeNameKey]; ok {
			nadaCache.MakeNameKeys[makeNameKey] = append(makeIDList, makeID)
		} else {
			nadaCache.MakeNameKeys[makeNameKey] = []int{makeID}
		}
	}
	nadaModelDetails := readNADAFile("ModelDetails", []string{"Company", "CompanyNum", "ModelYear", "ModelCat", "Model", "ModelNum", "ModelWeb", "Length", "Type", "Hull", "Beam", "Eng", "Weight", "Version", "NumberOfEngines", "HP"})
	detailIDs := map[int]bool{}
	for row := 0; row < nadaModelDetails.Len; row++ {
		// ModelNum should be unique integer
		modelNum := nadaModelDetails.get(row, "ModelNum")
		detailID, err := strconv.Atoi(modelNum)
		if err != nil {
			panic(errors.New("NADA ModelDetails has ModelNum \"" + modelNum + "\""))
		}
		logPrefix := "NADA ModelDetails where ModelNum=" + modelNum + ": "
		if _, ok := detailIDs[detailID]; ok {
			panic(errors.New(logPrefix + "duplicate"))
		}
		detailIDs[detailID] = true
		// CompanyNum should be integer matching MakeID
		companyNum := nadaModelDetails.get(row, "CompanyNum")
		makeID, err := strconv.Atoi(companyNum)
		if err != nil {
			panic(errors.New(logPrefix + "bad CompanyNum \"" + companyNum + "\""))
		}
		make := nadaCache.Makes[makeID]
		// Company must match Make.Name
		company := nadaModelDetails.get(row, "Company")
		if strings.ToUpper(company) != strings.ToUpper(make.Name) {
			panic(errors.New(logPrefix + "Company \"" + company + "\" instead of \"" + make.Name + "\""))
		}
		// ModelYear must be 4-digit year in Make.YearRanges
		modelYear := nadaModelDetails.get(row, "ModelYear")
		year, err := strconv.Atoi(modelYear)
		if err != nil {
			panic(errors.New(logPrefix + "bad ModelYear \"" + modelYear + "\""))
		}
		if !yearInRanges(year, make.YearRanges) {
			panic(errors.New(logPrefix + "ModelYear not in YearRanges"))
		}
		// Type must be a valid MakeDetails.Type enum
		detailType := textToPascalCase(nadaModelDetails.get(row, "Type"))
		if _, ok := validDetailTypes[detailType]; !ok {
			panic(errors.New(logPrefix + "bad Type \"" + detailType + "\""))
		}
		// Hull must be slash-separated list of valid materials
		hull := nadaModelDetails.get(row, "Hull")
		hullMaterials := []string{}
		if hull != "" {
			hullMaterials = strings.Split(hull, "/")
			for i, hullMaterial := range hullMaterials {
				if hullMaterial == "POLYESER" {
					hullMaterial = "POLYESTER"
				}
				if hullMaterial == "PP" {
					hullMaterial = "Polypropylene"
				}
				if hullMaterial == "POLY URETHENE" {
					hullMaterial = "POLYURETHENE"
				}
				hullMaterial = textToPascalCase(hullMaterial)
				if _, ok := validHullMaterials[hullMaterial]; !ok {
					panic(errors.New(logPrefix + "bad Hull \"" + hull + "\""))
				}
				hullMaterials[i] = hullMaterial
			}
		}
		// Eng must be empty or like "1<br>50 HP <br>Gasoline"
		eng := nadaModelDetails.get(row, "Eng")
		numberOfEngines := nadaModelDetails.get(row, "NumberOfEngines")
		hp := nadaModelDetails.get(row, "HP")
		enginePower := float64(mustBeInt(hp, logPrefix+"bad HP"))
		fuelType := "Unknown"
		if eng != "" {
			countHPFuel := validEng.FindStringSubmatch(eng)
			if len(countHPFuel) != 4 {
				panic(errors.New(logPrefix + "bad Eng \"" + eng + "\""))
			}
			if numberOfEngines != countHPFuel[1] {
				panic(errors.New(logPrefix + "NumberOfEngines \"" + numberOfEngines + "\" mismatches Eng \"" + eng + "\""))
			}
			if hp != countHPFuel[2] {
				enginePower, _ = strconv.ParseFloat(countHPFuel[2], 32)
				if hp != strconv.Itoa(int(enginePower)) {
					panic(errors.New(logPrefix + "HP \"" + hp + "\" mismatches Eng \"" + eng + "\""))
				}
			}
			fuelType = strings.ReplaceAll(countHPFuel[3], "Gasoline", "Gas")
			if _, ok := validFuelTypes[fuelType]; !ok {
				panic(errors.New(logPrefix + "bad fuel type \"" + fuelType + "\""))
			}
		}
		// add make details to cache
		make.Details = append(make.Details, MakeDetail{
			ID:            detailID,
			Year:          year,
			Series:        nadaModelDetails.get(row, "ModelCat"),
			Model:         nadaModelDetails.get(row, "Model"),
			Locomotion:    locomotionPerDetailType(detailType),
			Type:          detailType,
			Length:        mustBeFeet(nadaModelDetails.get(row, "Length"), logPrefix+"bad Length"),
			Beam:          mustBeFeet(nadaModelDetails.get(row, "Beam"), logPrefix+"bad Beam"),
			Weight:        float32(mustBeInt(nadaModelDetails.get(row, "Weight"), logPrefix+"bad Weight")),
			HullMaterials: hullMaterials,
			EngineCount:   mustBeInt(numberOfEngines, logPrefix+"bad NumberOfEngines"),
			EnginePower:   float32(enginePower),
			FuelType:      fuelType,
		})
	}
	nadaOptions := readNADAFile("Options", []string{"ModelYear", "OptionCat", "Description", "Version", "OptionNum"})
	optionIDs := map[int]bool{}
	for row := 0; row < nadaOptions.Len; row++ {
		// OptionNum should be unique integer
		optionNum := nadaOptions.get(row, "OptionNum")
		optionID, err := strconv.Atoi(optionNum)
		if err != nil {
			panic(errors.New("NADA Options has OptionNum \"" + optionNum + "\""))
		}
		logPrefix := "NADA Options where OptionNum=" + optionNum + ": "
		if _, ok := optionIDs[optionID]; ok {
			panic(errors.New(logPrefix + "duplicate"))
		}
		optionIDs[optionID] = true
		// ModelYear must be 4-digit year
		modelYear := nadaOptions.get(row, "ModelYear")
		year, err := strconv.Atoi(modelYear)
		if err != nil || year < 1980 || year > 2999 {
			panic(errors.New(logPrefix + "bad ModelYear \"" + modelYear + "\""))
		}
		// OptionCat is like "POWER BOAT:CANVAS" or "SAILBOAT:GALLEY"
		optionCat := nadaOptions.get(row, "OptionCat")
		powerCategory := validOptionCat.FindStringSubmatch(optionCat)
		if len(powerCategory) != 3 {
			panic(errors.New(logPrefix + "bad OptionCat \"" + optionCat + "\""))
		}
		power := 0
		switch powerCategory[1] {
		case "POWER BOAT":
			power = 1
		case "SAILBOAT":
			power = 0
		}
		category := powerCategory[2]
		description := nadaOptions.get(row, "Description")
		yearPowerOptions, ok := nadaCache.YearPowerOptions[year]
		if !ok {
			yearPowerOptions = map[int]map[string]map[string]int{}
			nadaCache.YearPowerOptions[year] = yearPowerOptions
		}
		powerOptions, ok := yearPowerOptions[power]
		if !ok {
			powerOptions = map[string]map[string]int{}
			yearPowerOptions[power] = powerOptions
		}
		options, ok := powerOptions[category]
		if !ok {
			options = map[string]int{}
			powerOptions[category] = options
		}
		options[description] = optionID
	}
	nadaCacheJSON, err := json.Marshal(&nadaCache)
	if err != nil {
		panic(errors.New("Can't turn makes into JSON: " + err.Error()))
	}
	if err := ioutil.WriteFile(path, nadaCacheJSON, 0644); err != nil {
		panic(errors.New("Can't write " + path + ": " + err.Error()))
	}
}

func yearListToYearRanges(yearList string) [][]int {
	result := [][]int{}
	if yearList != "" {
		years := strings.Split(yearList, ",")
		for _, yearString := range years {
			year, _ := strconv.Atoi(yearString)
			if len(result) == 0 || year > result[len(result)-1][1]+1 {
				result = append(result, []int{year, year})
			} else if year < result[len(result)-1][1]+1 {
				panic(errors.New("Bad year list: " + yearList))
			} else {
				result[len(result)-1][1] = year
			}
		}
		// add this year if previous year was active, just because NADA database might be outdated
		thisYear := now().Year()
		if result[len(result)-1][1] == thisYear-1 {
			result[len(result)-1][1] = thisYear
		}
	}
	return result
}

func yearInRanges(year int, yearRanges [][]int) bool {
	for _, startEnd := range yearRanges {
		if year >= startEnd[0] && year <= startEnd[1] {
			return true
		}
	}
	return false
}

// strip out common terms from manufacturer names and return it all upper case; i.e., "Angler Boats" and "Angler Boat Corp" both become "ANGLER", so we can match them
var makeNameStripPattern = regexp.MustCompile(`\s+(BOATS?|BOATWORKS|CO|COMPANY|CORP|CUSTOM|GROUP|INC\.?|INDUSTRIES|INFLATABLES|MARINE|MOTOR|OF AMERICA|POWERBOATS?|SAILING|USA|YACHTS?)\b`)

func getMakeNameKey(name string) string {
	return makeNameStripPattern.ReplaceAllString(strings.ToUpper(name), "")
}

// change "HI, I'M SOME TEXT" to "HiIMSomeText"
var firstLetterOrAcronyms = regexp.MustCompile(`\b(?i:VDR)\b|\b\w`)
var nonLetter = regexp.MustCompile(`\W`)

func textToPascalCase(s string) string {
	return nonLetter.ReplaceAllString(firstLetterOrAcronyms.ReplaceAllStringFunc(strings.ToLower(s), strings.ToUpper), "")
}

func locomotionPerDetailType(detailType string) string {
	if detailType == "Catamaran" || detailType == "MonohullSailboats" || detailType == "TrimaranBoats" {
		return "Sail"
	}
	return "Power"
}

var validFeet = regexp.MustCompile(`^(\d+)'( (\d+)")?$`)

// feet will convert `10' 9"` to 10.75
func mustBeFeet(s string, fatalMessage string) float32 {
	if s == "" {
		return 0
	}
	m := validFeet.FindStringSubmatch(s)
	if len(m) != 4 {
		panic(errors.New(fatalMessage + ": `" + s + "`"))
	}
	ft, _ := strconv.Atoi(m[1])
	in, _ := strconv.Atoi(m[3])
	return float32(ft) + float32(in)/12
}

func mustBeInt(s string, fatalMessage string) int {
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(errors.New(fatalMessage + ": `" + s + "`"))
	}
	return i
}

type table struct {
	Columns map[string]int
	Rows    [][]string
	Len     int
}

func readNADAFile(name string, requiredCols []string) table {
	path := "nada/" + name + ".csv"
	result, err := readCSVFile(path, requiredCols)
	if err != nil {
		panic(errors.New("Can't read " + path + ": " + err.Error()))
	}
	fmt.Println("Read " + path)
	return result
}

// readCSVFile reads a CSV file from path, makes sure all requireCols are in the header, and returns the column list, the rows, and a possible error
func readCSVFile(path string, requiredColumns []string) (table, error) {
	result := table{}
	csvFile, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer csvFile.Close()
	records, err := csv.NewReader(csvFile).ReadAll()
	if err != nil {
		return result, err
	}
	if len(records) == 0 {
		return result, errors.New("Empty csv file at " + path)
	}
	columns := records[0]
	columnsMap := map[string]int{}
	for i, columnName := range columns {
		columnsMap[columnName] = i
	}
	missingColumns := []string{}
	for _, requiredColumn := range requiredColumns {
		if _, ok := columnsMap[requiredColumn]; !ok {
			missingColumns = append(missingColumns, requiredColumn)
		}
	}
	if len(missingColumns) > 0 {
		return result, errors.New("Missing columns " + strings.Join(missingColumns, ", ") + " in file at " + path)
	}
	result.Columns = columnsMap
	result.Rows = records[1:]
	result.Len = len(result.Rows)
	return result, err
}

func (t *table) get(row int, column string) string {
	return t.Rows[row][t.Columns[column]]
}

// GetMakes gets makes
func GetMakes(req *Request, pub *Publication) *Response {
	if req.MakeID == 0 {
		return &Response{
			Makes: makes,
		}
	}
	make := LookupMake(req.Year, req.MakeID, req.MakeDetailID, "")
	if make == nil {
		return &Response{ErrorCode: "BadMakeID"}
	}
	result := map[int]*Make{}
	result[req.MakeID] = make
	return &Response{
		Makes: result,
	}
}

// LookupMake finds Make and its Details, either by:
// - make name and year, where make name doesn't have to be exact; i.e., "Angler Boats" instead of "Angler Boat Corp"
// - make id and optionally year and detail id
func LookupMake(year, makeID, makeDetailID int, makeName string) *Make {
	if makeName != "" {
		makeIDs, ok := nadaCache.MakeNameKeys[getMakeNameKey(makeName)]
		if !ok {
			return nil
		}
		numOfMakesHavingYear := 0
		for _, aMakeID := range makeIDs {
			aMake := nadaCache.Makes[aMakeID]
			if yearInRanges(year, aMake.YearRanges) {
				numOfMakesHavingYear++
				makeID = aMakeID
			}
		}
		if numOfMakesHavingYear != 1 { // either none or ambiguous
			return nil
		}
	}
	make, ok := nadaCache.Makes[makeID]
	if !ok {
		return nil
	}
	details := []MakeDetail{}
	if year != 0 {
		// if !yearInRanges(year, make.YearRanges) {
		// 	return &Response{Error: "Year inactive"}
		// }
		for _, detail := range make.Details {
			if detail.Year == year && (makeDetailID == 0 || makeDetailID == detail.ID) {
				if makeDetailID == detail.ID {
					power := 1
					if detail.Type == "Catamaran" || detail.Type == "MonohullSailboats" || detail.Type == "TrimaranBoats" {
						power = 0
					}
					detail.Options = nadaCache.YearPowerOptions[year][power]
				}
				details = append(details, detail)
			}
		}
	}
	return &Make{
		ID:         makeID,
		Name:       make.Name,
		Rank:       make.Rank,
		YearRanges: make.YearRanges,
		Details:    details,
	}
}
