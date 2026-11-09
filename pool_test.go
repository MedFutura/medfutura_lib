package medfutura_lib

import (
	"sync"
	"testing"
)

func TestPoolCompartilhado(t *testing.T) {
	var wg sync.WaitGroup
	dbs := make(chan any, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db, err := Pool(99, "TESTE")
			if err != nil {
				t.Error(err)
			}
			dbs <- db
		}()
	}
	wg.Wait()
	close(dbs)
	primeiro := <-dbs
	for db := range dbs {
		if db != primeiro {
			t.Fatal("Pool devolveu pools diferentes para o mesmo banco")
		}
	}
	if outro, _ := Pool(99, "OUTRO"); any(outro) == primeiro {
		t.Fatal("bancos diferentes compartilharam o pool")
	}
}
