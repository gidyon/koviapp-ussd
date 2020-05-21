package main

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

func (api *ussdAPIServer) riskAnalysis(userID string) (string, error) {
	lang, err := api.getUserLanguage(userID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user language")
	}

	risk, err := api.getRisk(userID)
	var riskIndex string
	switch {
	case risk > 10:
		if lang == eng {
			riskIndex = "HIGH"
		} else {
			riskIndex = "JUU"
		}
	case risk > 5 && risk < 10:
		if lang == eng {
			riskIndex = "MEDIUM"
		} else {
			riskIndex = "KATI"
		}
	default:
		if lang == eng {
			riskIndex = "LOW"
		} else {
			riskIndex = "CHINI"
		}
	}

	recommendations, err := api.getUserRecommendations(userID, lang)
	if err != nil {
		return "", err
	}
	if len(recommendations) > 3 {
		recommendations = recommendations[:3]
	}

	response := "END "

	switch lang {
	case eng:
		response += fmt.Sprintf("You have %s risk of getting COVID-19.\nObserve the following recommendations to reduce your risk\n", riskIndex)
		for index, recommendation := range recommendations {
			response += fmt.Sprintf("%d. %s\n", index+1, recommendation)
		}
		response += "\nTake the questionnaire on a daily basis in order to stay updated\n"
		response += "See you next time :)"
	default:
		response += fmt.Sprintf("Una hatari ya %s kupata COVID-19.\nZingatia maagizo uliyopewa ili kupunguza hatari yako\n", riskIndex)
		for index, recommendation := range recommendations {
			response += fmt.Sprintf("%d. %s\n", index+1, recommendation)
		}
		response += "\nFanya jaribi hili kila siku ndiposa ujikinge zaidi.\n"
		response += "Tutaonana wakati mwingine :)"
	}

	return response, nil
}

func (api *ussdAPIServer) getRisk(userID string) (int, error) {
	riskStr, err := api.cache.HGet(userID, "risk").Result()
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(riskStr)
}

func (api *ussdAPIServer) getUserRecommendations(userID, lang string) ([]string, error) {
	switch lang {
	case eng:
		return []string{"Wear mask", "Avoid congested places", "Keep social distance of 1.5 m"}, nil
	default:
		return []string{"Vaa Maski", "Epuka maeneo yenye watu wengi", "Zingatia umbali wa kijami wa 1.5 mita"}, nil
	}
}
