package model

type Model[T any] interface {
	DTO() *T
}

type MissionModel[T any] interface {
	DTO(missionName string) *T
}

func DTOList[N Model[T], T any](l []N) []*T {
	res := make([]*T, len(l))

	for i, x := range l {
		res[i] = x.DTO()
	}

	return res
}

func MissionDTOList[N MissionModel[T], T any](missionName string, l []N) []*T {
	res := make([]*T, len(l))

	for i, x := range l {
		res[i] = x.DTO(missionName)
	}

	return res
}