package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gidyon/micros/utils/healthcheck"

	"github.com/jinzhu/gorm"
	"google.golang.org/grpc/grpclog"

	"github.com/gidyon/micros"

	"github.com/Sirupsen/logrus"
	"github.com/gidyon/config"

	"github.com/go-redis/redis"
)

func main() {
	cfg, err := config.New()
	handleError(err)

	ctx := context.Background()

	service, err := micros.NewService(ctx, cfg, nil)
	handleError(err)

	ussdAPI := &ussdAPIServer{
		cache:  service.RedisClient(),
		sqlDB:  service.GormDB(),
		logger: service.Logger(),
	}

	service.AddEndpoint("/callbacks/ussd/screening", ussdAPI)

	// Health check endpoints
	service.AddEndpoint("/callbacks/ussd/screening/readyq", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service: service,
		Type:    healthcheck.ProbeReadiness,
	}))
	service.AddEndpoint("/callbacks/ussd/screening/liveq", healthcheck.RegisterProbe(&healthcheck.ProbeOptions{
		Service: service,
		Type:    healthcheck.ProbeLiveNess,
	}))

	handleError(service.Run(ctx))
}

func handleError(err error) {
	if err != nil {
		logrus.Fatalln(err)
	}
}

type ussdPayload struct {
	SessionID   string `json:"sessionId,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	NetworkCode string `json:"networkCode,omitempty"`
	ServiceCode string `json:"serviceCode,omitempty"`
	Text        string `json:"text,omitempty"`
}

type ussdAPIServer struct {
	cache  *redis.Client
	sqlDB  *gorm.DB
	logger grpclog.LoggerV2
}

func (api *ussdAPIServer) httpError(w http.ResponseWriter, userID, errMsg string, statusCode int) {
	api.deleteUserSession(userID)
	http.Error(w, "END "+errMsg, statusCode)
}

func (api *ussdAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get request
	ussd := &ussdPayload{}
	err := json.NewDecoder(r.Body).Decode(ussd)
	if err != nil {
		http.Error(w, "failed to decode request body", http.StatusInternalServerError)
		return
	}

	var response string

	switch {
	case ussd.Text == "":
		// Save user
		err = api.saveUser(ussd)
		if err != nil {
			http.Error(w, "failed to save user", http.StatusInternalServerError)
			return
		}

		response = "CON Welcome to KoviApp. Select language \n"
		response += "1. English \n"
		response += "2. Kiswahili"

	case ussd.Text == "1":
		err = api.setUserLanguage(ussd, eng)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed to set user language", http.StatusInternalServerError)
			return
		}
		response += "CON Select service you want to access \n"
		response += "1. Self-Screening for COVID-19"
		response += "2. View local hotlines"
	case ussd.Text == "2":
		err = api.setUserLanguage(ussd, swa)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed to set user language", http.StatusInternalServerError)
			return
		}
		response += "CON Changua huduma unachotaka kupata \n"
		response += "1. Kujichunguza dhidi ya COVID-19"
		response += "2. Tazama nambari za eneo"

	case ussd.Text == "1*2":
		response += "CON Type county name \n"
	case ussd.Text == "2*2":
		response += "CON Andika jina la kaunti \n"

	case strings.HasPrefix(ussd.Text, "1*2*") || strings.HasPrefix(ussd.Text, "2*2*"):
		if strings.HasPrefix(ussd.Text, "1*2*") {
			response = "Nambari za kupiga \n"
		} else {
			response = "Numbers to call \n"
		}
		hotlines, err := api.getHotlines(ussd.Text)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed to get hotlines", http.StatusInternalServerError)
			return
		}
		if len(hotlines) > 5 {
			hotlines = hotlines[:5]
		}

		for index, hotline := range hotlines {
			response += fmt.Sprintf("%d %s \n", index, hotline)
		}

		response = "END Thank you. Keep safe"

	case strings.HasPrefix(ussd.Text, "1*1"):
		response = "CON Welcome to KoviApp Self screenig. Provide honest response \n"
		response += "How old are you \n"
		response += "1. 0 - 15 years \n"
		response += "2. 15 - 25 years \n"
		response += "3. 25 - 40 years \n"
		response += "4. 40 - 60 years \n"
		response += "5. Above 60 years \n"
	case strings.HasPrefix(ussd.Text, "2*1"):
		response = "CON Karibu kwenye KoviApp uchunguzi wa kibinafsi. Toa mwitikio wa kweli \n"
		response += "Una miaka mingapi \n"
		response += "1. Miaka 0 - 15 \n"
		response += "2. Miaka 15 - 25 \n"
		response += "3. Miaka 25 - 40 \n"
		response += "4. Miaka 40 - 60 \n"
		response += "5. Miaka zaidi ya 60 \n"

		// Age Section
	case len(ussd.Text) == 5 && (strings.HasPrefix(ussd.Text, "1*1*") || strings.HasPrefix(ussd.Text, "2*1*")) &&
		strings.Count(ussd.Text, "*") == 2:
		switch {
		case strings.HasSuffix(ussd.Text, "*1"):
			err = api.saveUserAge(ussd.SessionID, "0 - 15", 1)
		case strings.HasSuffix(ussd.Text, "*2"):
			err = api.saveUserAge(ussd.SessionID, "15 - 25", 1)
		case strings.HasSuffix(ussd.Text, "*3"):
			err = api.saveUserAge(ussd.SessionID, "25 - 40", 2)
		case strings.HasSuffix(ussd.Text, "*4"):
			err = api.saveUserAge(ussd.SessionID, "40 - 60", 2)
		case strings.HasSuffix(ussd.Text, "*5"):
			err = api.saveUserAge(ussd.SessionID, "Above 60", 3)
		}
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed to save user age", http.StatusInternalServerError)
			return
		}
		response, err = api.responseForCases(ussd.SessionID)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed to create response for cases", http.StatusInternalServerError)
			return
		}

		// Cases section
	case len(ussd.Text) == 7 && (strings.HasPrefix(ussd.Text, "1*1*") || strings.HasPrefix(ussd.Text, "2*1*")) &&
		strings.Count(ussd.Text, "*") == 3:
		switch {
		case strings.HasSuffix(ussd.Text, "*1"):
			err = api.saveUserCases(ussd.SessionID, "More than 100", 2)
		case strings.HasSuffix(ussd.Text, "*2"):
			err = api.saveUserCases(ussd.SessionID, "Less than 100", 1)
		case strings.HasSuffix(ussd.Text, "*3"):
			err = api.saveUserCases(ussd.SessionID, "Not known", 1)
		}
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed to save cases", http.StatusInternalServerError)
			return
		}

		response, err = api.responseForContact(ussd.SessionID)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed create response for contact", http.StatusInternalServerError)
			return
		}

		// Contact with suspected or confirmed COVID-19 patient
	case len(ussd.Text) == 9 && (strings.HasPrefix(ussd.Text, "1*1*") || strings.HasPrefix(ussd.Text, "2*1*")) &&
		strings.Count(ussd.Text, "*") == 4:
		switch {
		case strings.HasSuffix(ussd.Text, "*1"):
			err = api.saveUserContactStatus(ussd.SessionID, "yes", 3)
		case strings.HasSuffix(ussd.Text, "*2"):
			err = api.saveUserContactStatus(ussd.SessionID, "no", 1)
		case strings.HasSuffix(ussd.Text, "*3"):
			err = api.saveUserContactStatus(ussd.SessionID, "uknown", 1)
		}
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed to save cases", http.StatusInternalServerError)
			return
		}

		response, err = api.responseForHowContactHappened(ussd.SessionID)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed create response for contact", http.StatusInternalServerError)
			return
		}

		// How contact happenned
	case (strings.HasPrefix(ussd.Text, "1*1*") || strings.HasPrefix(ussd.Text, "2*1*")) &&
		strings.Count(ussd.Text, "*") == 5:
		last := strings.LastIndex(ussd.Text, "*")
		if last > 0 {
			answers := strings.Split(ussd.Text[last:], ",")
			if len(answers) == 0 {
				answers = strings.Split(ussd.Text[last:], " ")
			}

			for _, answer := range answers {
				switch strings.TrimSpace(answer) {
				case "1":
					err = api.saveHowContactHappenedSelection(ussd.SessionID, "working together", 1)
				case "2":
					err = api.saveHowContactHappenedSelection(ussd.SessionID, "face to face contact within 1 meter", 2)
				case "3":
					err = api.saveHowContactHappenedSelection(ussd.SessionID, "travelling together", 1)
				case "4":
					err = api.saveHowContactHappenedSelection(ussd.SessionID, "living in the same environment", 2)
				case "5":
					err = api.saveHowContactHappenedSelection(ussd.SessionID, "health care associated exposure", 2)
				case "6":
					err = api.saveHowContactHappenedSelection(ussd.SessionID, "other", 1)
				case "7":
					err = api.saveHowContactHappenedSelection(ussd.SessionID, "none", 0)
				}

				if err != nil {
					api.httpError(w, ussd.SessionID, "failed to save how contact happened", http.StatusInternalServerError)
					return
				}
			}
		}

		response, err = api.responseForSymptoms(ussd.SessionID)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed create response for symptoms", http.StatusInternalServerError)
			return
		}

	case (strings.HasPrefix(ussd.Text, "1*1*") || strings.HasPrefix(ussd.Text, "2*1*")) &&
		strings.Count(ussd.Text, "*") == 6:
		last := strings.LastIndex(ussd.Text, "*")
		if last > 0 {
			answers := strings.Split(ussd.Text[last:], ",")
			if len(answers) == 0 {
				answers = strings.Split(ussd.Text[last:], " ")
			}

			for _, answer := range answers {
				switch strings.TrimSpace(answer) {
				case "1":
					err = api.saveSymptomsSelection(ussd.SessionID, "difficulty in breathing", 1)
				case "2":
					err = api.saveSymptomsSelection(ussd.SessionID, "cough", 1)
				case "3":
					err = api.saveSymptomsSelection(ussd.SessionID, "fatigue", 1)
				case "4":
					err = api.saveSymptomsSelection(ussd.SessionID, "fever", 1)
				case "5":
					err = api.saveSymptomsSelection(ussd.SessionID, "none of the above", 0)
				}

				if err != nil {
					api.httpError(w, ussd.SessionID, "failed save symptoms selection", http.StatusInternalServerError)
					return
				}
			}
		}

		response, err = api.responseForIllness(ussd.SessionID)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed create response for illness", http.StatusInternalServerError)
			return
		}

	case (strings.HasPrefix(ussd.Text, "1*1*") || strings.HasPrefix(ussd.Text, "2*1*")) &&
		strings.Count(ussd.Text, "*") == 7:
		last := strings.LastIndex(ussd.Text, "*")
		if last > 0 {
			answers := strings.Split(ussd.Text[last:], ",")
			if len(answers) == 0 {
				answers = strings.Split(ussd.Text[last:], " ")
			}

			for _, answer := range answers {
				switch strings.TrimSpace(answer) {
				case "1":
					err = api.saveIllnessSelection(ussd.SessionID, "diabetes", 1)
				case "2":
					err = api.saveIllnessSelection(ussd.SessionID, "asthmatic", 2)
				case "3":
					err = api.saveIllnessSelection(ussd.SessionID, "cancer", 1)
				case "4":
					err = api.saveIllnessSelection(ussd.SessionID, "hyper tension", 2)
				case "5":
					err = api.saveIllnessSelection(ussd.SessionID, "tuberclosis", 2)
				case "6":
					err = api.saveIllnessSelection(ussd.SessionID, "respiratory illness", 2)
				case "7":
					err = api.saveIllnessSelection(ussd.SessionID, "none of the above", 0)
				}

				if err != nil {
					api.httpError(w, ussd.SessionID, "failed save illness selection", http.StatusInternalServerError)
					return
				}
			}
		}

		// Calculate risk
		response, err = api.riskAnalysis(ussd.SessionID)
		if err != nil {
			api.httpError(w, ussd.SessionID, "failed create risk analysis", http.StatusInternalServerError)
			return
		}

	default:

	}
	if err != nil {
		api.httpError(w, ussd.SessionID, "failed to get hotlines", http.StatusInternalServerError)
		return
	}
}

func (api *ussdAPIServer) getHotlines(county string) ([]string, error) {
	return []string{"0716282395", "07453423"}, nil
}
