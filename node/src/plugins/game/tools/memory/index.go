package game_memory

import (
	"context"
	"fmt"
	"kasper/src/shell/api/model"
	"kasper/src/shell/layer1/adapters"
	"log"

	"github.com/redis/go-redis/v9"
)

func BuildLeaderboardKey(gameKey string, level string) string {
	return fmt.Sprintf("board-%s-%s", gameKey, level)
}

func Add(cache adapters.ICache, userId string, score float64, gameKey string, level string) {
	var leaderboard = BuildLeaderboardKey(gameKey, level)
	oldValue, err2 := cache.Infra().(*redis.Client).ZScore(context.Background(), leaderboard, userId).Result()
	if err2 != nil {
		fmt.Println(err2)
		oldValue = 0
	}
	var newVAlue = score + oldValue
	cache.Infra().(*redis.Client).ZAdd(context.Background(), leaderboard, redis.Z{Member: userId, Score: newVAlue})
}

func Replace(cache adapters.ICache, userId string, score float64, gameKey string, level string) {
	var leaderboard = BuildLeaderboardKey(gameKey, level)
	cache.Infra().(*redis.Client).ZAdd(context.Background(), leaderboard, redis.Z{Member: userId, Score: score})
	log.Println(leaderboard, score)
}

func Kickout(cache adapters.ICache, gameKey string, level string, userId string) {
	var leaderboard = BuildLeaderboardKey(gameKey, level)
	_, err := cache.Infra().(*redis.Client).ZRem(context.Background(), leaderboard, userId).Result()
	if err != nil {
		fmt.Println(err)
	}
}

func HumanRank(cache adapters.ICache, userId string, gameKey string, level string, asc bool) int64 {
	var leaderboard = BuildLeaderboardKey(gameKey, level)
	if asc {
		rank, err := cache.Infra().(*redis.Client).ZRevRank(context.Background(), leaderboard, userId).Result()
		if err != nil {
			fmt.Println(err)
			return -1
		}
		return rank
	} else {
		rank, err := cache.Infra().(*redis.Client).ZRank(context.Background(), leaderboard, userId).Result()
		if err != nil {
			fmt.Println(err)
			return -1
		}
		return rank
	}
}

func IncrAndReturn(cache adapters.ICache, userId string, gameKey string, level string, addingVal float64) float64 {
	ctx := context.Background()
	var leaderboard = BuildLeaderboardKey(gameKey, level)
	_, err := cache.Infra().(*redis.Client).ZIncrBy(ctx, leaderboard, addingVal, userId).Result()
	if err != nil {
		fmt.Println(err)
	}
	score, err2 := cache.Infra().(*redis.Client).ZScore(ctx, leaderboard, userId).Result()
	if err2 != nil {
		fmt.Println(err2)
	}
	return score
}

func FindSCore(cache adapters.ICache, userId string, gameKey string, level string) float64 {
	ctx := context.Background()
	var leaderboard = BuildLeaderboardKey(gameKey, level)
	score, err2 := cache.Infra().(*redis.Client).ZScore(ctx, leaderboard, userId).Result()
	if err2 != nil {
		fmt.Println(err2)
	}
	return score
}

func TotalCount(cache adapters.ICache, gameKey string, level string) int64 {
	res, err := cache.Infra().(*redis.Client).ZCard(context.Background(), BuildLeaderboardKey(gameKey, level)).Result()
	if err != nil {
		fmt.Println(err)
		return 0
	}
	return res
}

func Top10Players(cache adapters.ICache, gameKey string, level string, asc bool) [10]TopPlayer {
	var topPlayers = []redis.Z{}
	if asc {
		topPs, err := cache.Infra().(*redis.Client).ZRevRangeByScoreWithScores(context.Background(), BuildLeaderboardKey(gameKey, level), &redis.ZRangeBy{
			Min:    "-inf",
			Max:    "+inf",
			Offset: 0,
			Count:  10,
		}).Result()
		if err != nil {
			fmt.Println(err)
			return [10]TopPlayer{}
		}
		topPlayers = topPs
	} else {
		topPs, err := cache.Infra().(*redis.Client).ZRangeByScoreWithScores(context.Background(), BuildLeaderboardKey(gameKey, level), &redis.ZRangeBy{
			Min:    "-inf",
			Max:    "+inf",
			Offset: 0,
			Count:  10,
		}).Result()
		if err != nil {
			fmt.Println(err)
			return [10]TopPlayer{}
		}
		topPlayers = topPs
	}
	var tps = [10]TopPlayer{}
	for index, v := range topPlayers {
		// if v.Score > 0 {
		tps[index] = TopPlayer{UserId: v.Member.(string), Score: v.Score}
		// }
	}
	return tps
}

type TopPlayer struct {
	UserId string
	Score  float64
}

func TopPlayers(cache adapters.ICache, gameKey string, level string, asc bool) ([100]TopPlayer, int) {
	var key = BuildLeaderboardKey(gameKey, level)
	var topPlayers = []redis.Z{}
	if asc {
		topPs, err := cache.Infra().(*redis.Client).ZRevRangeByScoreWithScores(context.Background(), key, &redis.ZRangeBy{
			Min:    "-inf",
			Max:    "+inf",
			Offset: 0,
			Count:  100,
		}).Result()
		if err != nil {
			log.Println(err)
			return [100]TopPlayer{}, 0
		}
		topPlayers = topPs
	} else {
		topPs, err := cache.Infra().(*redis.Client).ZRangeByScoreWithScores(context.Background(), key, &redis.ZRangeBy{
			Min:    "-inf",
			Max:    "+inf",
			Offset: 0,
			Count:  100,
		}).Result()
		if err != nil {
			log.Println(err)
			return [100]TopPlayer{}, 0
		}
		topPlayers = topPs
	}
	var tps = [100]TopPlayer{}
	for index, v := range topPlayers {
		//if v.Score > 0 {
		tps[index] = TopPlayer{UserId: v.Member.(string), Score: v.Score}
		//}
	}
	count, err := cache.Infra().(*redis.Client).ZCard(context.Background(), key).Result()
	if err != nil {
		fmt.Println(err)
		return [100]TopPlayer{}, 0
	}
	if count >= 100 {
		return tps, 100
	} else {
		return tps, int(count)
	}
}

func LoadPlayersIntoMemory(storage adapters.IStorage, cache adapters.ICache, gameKey string, levels []string, scoreKey string) {
	for i := 0; i < len(levels); i++ {
		var leaderboard = BuildLeaderboardKey(gameKey, levels[i])
		var scores = []redis.Z{}
		type playerdata struct {
			UserId string
			Score  float64
		}
		playerArr := []playerdata{}
		err := storage.Db().Model(&model.User{}).Select("id as user_id, " + adapters.BuildJsonFetcher("metadata", gameKey+".board."+levels[i]+"."+scoreKey) + " as score").Where("1 = 1").Find(&playerArr).Error
		if err != nil {
			log.Println(err)
			return
		}
		for _, player := range playerArr {
			if player.Score > 0 {
				ps := redis.Z{Member: player.UserId, Score: player.Score}
				scores = append(scores, ps)
				cache.Infra().(*redis.Client).ZAdd(context.Background(), leaderboard, ps)
			}
		}
	}
}

func ClearPlayersFromMemory(cache adapters.ICache, gameKey string, level string) {
	var leaderboard = BuildLeaderboardKey(gameKey, level)
	_, err := cache.Infra().(*redis.Client).Del(context.Background(), leaderboard).Result()
	if err != nil {
		fmt.Println(err)
	}
}
