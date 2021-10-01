package main

import (
	"github.com/goccy/go-json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// Maps category in the long way
	category = make(map[string]string)
)

func init() {
	category["EC"] = "EuroCity"
	category["EN"] = "EuroNight"
	category["TGV"] = "Train à Grande Vitesse"
	category["EXP"] = "Espresso "
	category["FR"] = "Frecciarossa Alta Velocità"
	category["FA"] = "Frecciargento Alta Velocità"
	category["ITA"] = "italo"
	category["FB"] = "Frecciabianca"
	category["IC"] = "InterCity"
	category["ICN"] = "InterCity notte"
	category["REG"] = "Regionale"
	category["RV"] = "Regionale Veloce"
	category["S"] = "Suburbano"
	category["RE"] = "RegioExpress"
	category["MXP"] = "Malpensa Express"
	category["SFM"] = "Servizio Ferroviario Metropolitano"
	category["M"] = "Treno metropolitano"
	category["ACC"] = "Accelerato"
	category["D"] = "Diretto"
	category["DD"] = "Direttissimo"

}

// Search where the given trainID starts
func searchAndGetTrain(trainID string) string {

	resp, err := http.Get("http://www.viaggiatreno.it/viaggiatrenonew/resteasy/viaggiatreno/cercaNumeroTrenoTrenoAutocomplete/" + trainID)
	if err != nil {
		return ""
	}

	body, _ := ioutil.ReadAll(resp.Body)

	out := string(body)
	_ = resp.Body.Close()

	if strings.TrimSpace(out) == "" {
		return ""
	}

	foo := strings.Split(out, "|")

	if len(foo) >= 1 {
		combinato := strings.Split(foo[1], "-")

		if len(combinato) >= 1 {
			return getTrain(strings.TrimSpace(combinato[1]) + "/" + combinato[0])
		}
	}

	return ""
}

// Returns text for a given train
func getTrain(idStazioneTreno string) string {

	var (
		pls               = true
		stazioni, binario string
		ritardo           int
		ora               time.Time
		treno             = treno{}
		out               string
	)

	res, err := http.Get("http://www.viaggiatreno.it/viaggiatrenonew/resteasy/viaggiatreno/andamentoTreno/" + idStazioneTreno + "/" + midnight())
	if err != nil {
		return ""
	}

	err = json.NewDecoder(res.Body).Decode(&treno)
	_ = res.Body.Close()
	if err != nil {
		return ""
	}

	for _, stazione := range treno.Fermate {
		if stazione.Stazione != treno.Origine && stazione.Stazione != treno.Destinazione && !pls {
			stazioni += stazione.Stazione + ","
		}

		if pls && stazione.Effettiva == nil {
			binario = stazione.BinarioProgrammatoPartenzaDescrizione
			ora = time.Unix(stazione.Programmata/1000, 0)
			ora = ora.Add(time.Minute * time.Duration(ritardo))
			pls = false
		} else {
			ritardo = stazione.Ritardo
		}

	}

	stazioni = strings.TrimSuffix(stazioni, ",") + "."

	out = "Il treno " + category[strings.Split(strings.TrimSpace(treno.CompNumeroTreno), " ")[0]] + ", " + formatNumber(treno.NumeroTreno) + ", di trenitalia, proveniente da " + treno.Origine + " ,e diretto a " + treno.Destinazione + ", delle ore " + ora.Format("15:04") + ", e' in arrivo al binario " + binario + "! Attenzione! Allontanarsi dalla linea gialla!"

	if stazioni != "." {
		out += "Ferma a: " + stazioni
	}

	return out
}

// Returns strange value (seems to be the midnight of the current day multiplied by 1000) that the API needs at the end for some calls. Don't ask, I didn't.
func midnight() string {
	t := time.Now()
	return strconv.FormatInt(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Unix()*1000, 10)
}

// Spaces out the train number
func formatNumber(nTrain int) string {
	out := ""
	nTrainRune := []rune(strconv.Itoa(nTrain))

	switch len(nTrainRune) {
	case 0, 1, 2, 3:
		for i := range nTrainRune {
			out += string(nTrainRune[i]) + ", "
		}

	case 4:
		out = string(nTrainRune[0]) + string(nTrainRune[1]) + ", " + string(nTrainRune[2]) + string(nTrainRune[3])

	case 5:
		out = string(nTrainRune[0]) + string(nTrainRune[1]) + ", " + string(nTrainRune[2]) + "-" + string(nTrainRune[3]) + "-" + string(nTrainRune[4])
	}

	return out
}
