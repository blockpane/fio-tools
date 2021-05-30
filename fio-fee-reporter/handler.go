package ffr

import (
	"github.com/blockpane/fio-tools/fio-fee-reporter/models"
	ops "github.com/blockpane/fio-tools/fio-fee-reporter/restapi/operations"
	"github.com/fioprotocol/fio-go/eos"
	"github.com/go-openapi/runtime/middleware"
)

func fee(usd bool) middleware.Responder {
	if !state.ready() {
		return &ops.GetFeeServiceUnavailable{
			Payload: &models.Error{
				Code:    503,
				Message: "data is not up to date, please try again later",
			},
		}
	}
	payload := &ops.GetFeeOK{Payload: make([]*models.Price, 0)}
	for i := range state.Fees {
		price := float64(state.Fees[i].SufAmount) / 1_000_000_000.0
		if usd {
			price = price * state.Price
		}
		payload.Payload = append(payload.Payload, &models.Price{
			EndPoint: &state.Fees[i].EndPoint,
			Price:    &price,
		})
	}
	return payload
}

func Fee(ops.GetFeeParams) middleware.Responder {
	return fee(false)
}

func FeeUsd(ops.GetFeeUsdParams) middleware.Responder {
	return fee(true)
}

func FeeVotesFeevoteProducer(params ops.GetFeeVotesFeevoteProducerParams) middleware.Responder {
	if _, e := eos.StringToName(params.Producer); e != nil || len(params.Producer) != 12 {
		return &ops.GetFeeVotesFeevoteProducerBadRequest{
			Payload: &models.Error{
				Code:    400,
				Message: "invalid account",
			},
		}
	}
	if !state.ready() {
		return &ops.GetFeeVotesFeevoteProducerServiceUnavailable{
			Payload: &models.Error{
				Code:    503,
				Message: "data is not up to date, please try again later",
			},
		}
	}
	// FIXME
	return middleware.NotImplemented("operation ops.GetFeeVotesProducerUsd has not yet been implemented")
}

func FeeVotesMultiplierProducer(params ops.GetFeeVotesMultiplierProducerParams) middleware.Responder {
	if _, e := eos.StringToName(params.Producer); e != nil || len(params.Producer) != 12 {
		return &ops.GetFeeVotesMultiplierProducerBadRequest{
			Payload: &models.Error{
				Code:    400,
				Message: "invalid account",
			},
		}
	}
	if !state.ready() {
		return &ops.GetFeeVotesMultiplierProducerServiceUnavailable{
			Payload: &models.Error{
				Code:    503,
				Message: "data is not up to date, please try again later",
			},
		}
	}
	var vote float64
	for i := range state.FeeVoters {
		if string(state.FeeVoters[i].BlockProducerName) == params.Producer {
			vote = state.FeeVoters[i].FeeMultiplier
			if vote == 0 {
				return &ops.GetFeeVotesMultiplierProducerNotFound{
					Payload: &models.Error{
						Code:    404,
						Message: "multiplier not found",
					},
				}
			}
			return &ops.GetFeeVotesMultiplierProducerOK{Payload: &ops.GetFeeVotesMultiplierProducerOKBody{Multiplier: vote}}
		}
	}
	return &ops.GetFeeVotesMultiplierProducerNotFound{
		Payload: &models.Error{
			Code:    404,
			Message: "producer not found",
		},
	}
}

func getProducerVotes(bp string, usd bool) middleware.Responder {
	if _, e := eos.StringToName(bp); e != nil || len(bp) != 12 {
		return &ops.GetFeeVotesFeevoteProducerBadRequest{
			Payload: &models.Error{
				Code:    400,
				Message: "invalid account",
			},
		}
	}
	if !state.ready() {
		return &ops.GetFeeVotesFeevoteProducerServiceUnavailable{
			Payload: &models.Error{
				Code:    503,
				Message: "data is not up to date, please try again later",
			},
		}
	}
	for i := range state.FeeVotes {
		if string(state.FeeVotes[i].BlockProducerName) == bp {
			var multiplier float64
			for _, mult := range state.FeeVoters {
				if string(mult.BlockProducerName) == bp {
					multiplier = mult.FeeMultiplier
					break
				}
			}
			if multiplier == 0 {
				return &ops.GetFeeVotesFeevoteProducerNotFound{
					Payload: &models.Error{
						Code:    404,
						Message: "multiplier not found, unable to calculate effective fee",
					},
				}
			}
			payload := &ops.GetFeeVotesProducerOK{Payload: make([]*models.Price, 0)}
			for _, v := range state.FeeVotes[i].FeeVotes {
				endpoint := v.EndPoint // dereference
				price := (float64(v.Value) / 1_000_000_000.0) * multiplier
				if usd {
					price = price * state.Price
				}
				payload.Payload = append(payload.Payload, &models.Price{
					EndPoint: &endpoint,
					Price:    &price,
				})
			}
			return payload
		}
	}
	return &ops.GetFeeVotesFeevoteProducerNotFound{
		Payload: &models.Error{
			Code:    404,
			Message: "producer not found",
		},
	}
}

func FeeVotesProducer(params ops.GetFeeVotesProducerParams) middleware.Responder {
	return getProducerVotes(params.Producer, false)
}

func FeeVotesProducerUsd(params ops.GetFeeVotesProducerUsdParams) middleware.Responder {
	return getProducerVotes(params.Producer, true)
}

func GetPrice(ops.GetPriceParams) middleware.Responder {
	if !state.ready() {
		return &ops.GetPriceServiceUnavailable{
			Payload: &models.Error{
				Code:    503,
				Message: "data is not up to date, please try again later",
			},
		}
	}
	return &ops.GetPriceOK{
		Payload: &ops.GetPriceOKBody{
			Price: state.Price,
		},
	}
}
