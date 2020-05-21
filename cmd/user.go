package main

import (
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

const (
	eng = "en"
	swa = "sw"
)

func (api *ussdAPIServer) saveUser(ussd *ussdPayload) error {
	return api.cache.HMSet(ussd.SessionID, "phone", ussd.PhoneNumber, "sessionId", ussd.PhoneNumber).Err()
}

func (api *ussdAPIServer) deleteUserSession(userID string) error {
	return api.cache.Del(userID).Err()
}

func (api *ussdAPIServer) responseForSelectService(ussd *ussdPayload) (string, error) {
	lang, err := api.getUserLanguage(ussd.SessionID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user language")
	}

	var response string
	switch lang {
	case eng:
		response += "CON Select service you want to access\n"
		response += "1. Self-Screening\n"
		response += "2. Local hotlines\n"
	default:
		response += "CON Changua huduma unataka kupata\n"
		response += "1. Kujichunguza"
		response += "2. Nambari ya eneo"
	}

	return response, nil
}

func (api *ussdAPIServer) setUserLanguage(ussd *ussdPayload, language string) error {
	err := api.cache.HSet(ussd.SessionID, "lang", language).Err()
	if err != nil {
		return err
	}
	return nil
}

func (api *ussdAPIServer) getUserLanguage(userID string) (string, error) {
	return api.cache.HGet(userID, "lang").Result()
}

func (api *ussdAPIServer) getUserFromSession(sessionID string) (map[string]string, error) {
	return api.cache.HGetAll(sessionID).Result()
}

func (api *ussdAPIServer) saveUserAge(userID, ageBracket string, score int) error {
	err := api.cache.HSet(userID, "ageBracket", ageBracket).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save user age")
	}
	err = api.cache.HIncrBy(userID, "risk", int64(score)).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save user score")
	}

	return nil
}

func (api *ussdAPIServer) responseForCases(userID string) (string, error) {
	lang, err := api.getUserLanguage(userID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user language")
	}

	var response = "CON "

	switch lang {
	case eng:
		response += "Have there been any case of COVID-19 in your area?\n"
		response += "1. More than 100 cases\n"
		response += "2. Less than 100\n"
		response += "3. Not known"
	default:
		response += "Kumekuwa na kesi yoyote ya COVID-19 katika eneo lako?\n"
		response += "1. Zaidi ya kesi 100\n"
		response += "2. Chini ya kesi 100\n"
		response += "3. Haijulikani"
	}

	return response, nil
}

func (api *ussdAPIServer) saveUserCases(userID, cases string, score int) error {
	err := api.cache.HSet(userID, "aerialCases", cases).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save cases")
	}
	err = api.cache.HIncrBy(userID, "risk", int64(score)).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save user score")
	}

	return nil
}

func (api *ussdAPIServer) responseForContact(userID string) (string, error) {
	lang, err := api.getUserLanguage(userID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user language")
	}

	var response string

	switch lang {
	case eng:
		response = "CON Have you been in contact with a suspected or confiimed COVID-19 case?\n"
		response += "1. Yes\n"
		response += "2. No\n"
		response += "3. Not Sure"
	default:
		response = "CON Je! Ushawai karibiana na mgonjwa anayeshukiwa au aliyethibitika kuwa na COVID-19?\n"
		response += "1. Ndio\n"
		response += "2. Hapana\n"
		response += "3. Sina hakika"
	}

	return response, nil
}

func (api *ussdAPIServer) saveUserContactStatus(userID, answer string, score int) error {
	err := api.cache.HSet(userID, "contactWithCOVID", answer).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save contact profile")
	}

	err = api.cache.HIncrBy(userID, "risk", int64(score)).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save user score")
	}

	return nil
}

func (api *ussdAPIServer) responseForHowContactHappened(userID string) (string, error) {
	lang, err := api.getUserLanguage(userID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user language")
	}

	var response string

	switch lang {
	case eng:
		response = "CON Have did the contact happened?\n"
		response += "1. Working together\n"
		response += "2. Face to face contact\n"
		response += "3. Travelling together\n"
		response += "4. Living in same environment\n"
		response += "5. Healthcare associated exposure\n"
		response += "6. None\n"
		response += "Use commas for multiple answers"
	default:
		response = "CON Je! Mapatano yalikuwaje?\n"
		response += "1. Kufanya kazi pamoja\n"
		response += "2. Uso wa uso\n"
		response += "3. Kusafiri pamoja\n"
		response += "4. Kuishi katika mazingira sawa\n"
		response += "5. Kupeana matibabu\n"
		response += "6. Hakuna\n"
		response += "Tumia comma kutenganisha majibu"
	}

	return response, nil
}

func (api *ussdAPIServer) saveHowContactHappenedSelection(userID, answer string, score int) error {
	values, err := api.cache.HGet(userID, "contacts").Result()
	if err != nil && err != redis.Nil {
		return errors.Wrap(err, "failed to get contact profile")
	}

	err = api.cache.HIncrBy(userID, "risk", int64(score)).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save user score")
	}

	values += answer
	return api.cache.HSet(userID, "contacts", values).Err()
}

func (api *ussdAPIServer) responseForSymptoms(userID string) (string, error) {
	lang, err := api.getUserLanguage(userID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user language")
	}

	var response string

	switch lang {
	case eng:
		response = "CON Do you have any of the following symptoms? \n"
		response += "1. Difficulty in breathing \n"
		response += "2. Cough \n"
		response += "3. Tiredness/Fatigue \n"
		response += "4. Fever \n"
		response += "5. None of the above"
	default:
		response = "CON Je! Una dalili zifuatazo? \n"
		response += "1. Ugumu wa kupumua \n"
		response += "2. Kikohozi \n"
		response += "3. Uchovu \n"
		response += "4. Homa \n"
		response += "5. Hakuna"
	}

	return response, nil
}

func (api *ussdAPIServer) saveSymptomsSelection(userID, answer string, score int) error {
	values, err := api.cache.HGet(userID, "symptoms").Result()
	if err != nil && err != redis.Nil {
		return errors.Wrap(err, "failed to save contact profile")
	}

	err = api.cache.HIncrBy(userID, "risk", int64(score)).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save user score")
	}

	values += answer
	return api.cache.HSet(userID, "symptoms", values).Err()
}

func (api *ussdAPIServer) responseForIllness(userID string) (string, error) {
	lang, err := api.getUserLanguage(userID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get user language")
	}

	var response string

	switch lang {
	case eng:
		response = "CON Do you have any of the following?\n"
		response += "1. Diabetes\n"
		response += "2. Asthmatic\n"
		response += "3. Cancer\n"
		response += "4. Hyper Tension\n"
		response += "5. Tuberclosis\n"
		response += "6. Respiratory illness\n"
		response += "7. None of the above"
	default:
		response = "CON Je! Unaugua yoyote yafuatayo?\n"
		response += "1. Ugonjwa wa sukari\n"
		response += "2. Pumu\n"
		response += "3. Saratani\n"
		response += "4. Shinikizo la damu\n"
		response += "5. Kifua kikuu\n"
		response += "6. Ugonjwa wa kupumua\n"
		response += "7. Hakuna yaliyo hapo juu"
	}

	return response, nil
}

func (api *ussdAPIServer) saveIllnessSelection(userID, answer string, score int) error {
	values, err := api.cache.HGet(userID, "illness").Result()
	if err != nil && err != redis.Nil {
		return errors.Wrap(err, "failed to save illness profile")
	}

	err = api.cache.HIncrBy(userID, "risk", int64(score)).Err()
	if err != nil {
		return errors.Wrap(err, "failed to save user score")
	}

	values += answer
	return api.cache.HSet(userID, "illness", values).Err()
}
