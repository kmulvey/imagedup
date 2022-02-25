package main

// shutdown gracefully shuts everything down and stores caches for next time
func shutdown(cache *hashCache) error {

	var err = cache.Persist(hashCacheFile)
	if err != nil {
		return err
	}

	return cache.Persist(hashCacheFile)
}
