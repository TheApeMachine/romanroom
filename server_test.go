package main

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAgenticMemoryServer(t *testing.T) {
	Convey("Given a server configuration", t, func() {
		config := DefaultServerConfig()
		
		Convey("When creating a new AgenticMemoryServer", func() {
			server, err := NewAgenticMemoryServer(config)
			
			Convey("Then it should create successfully", func() {
				So(err, ShouldBeNil)
				So(server, ShouldNotBeNil)
				So(server.GetConfig(), ShouldEqual, config)
				So(server.GetServer(), ShouldNotBeNil)
				So(server.IsRunning(), ShouldBeFalse)
			})
		})
		
		Convey("When creating a server with nil config", func() {
			server, err := NewAgenticMemoryServer(nil)
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(server, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "server config cannot be nil")
			})
		})
		
		Convey("When creating a server with invalid config", func() {
			invalidConfig := DefaultServerConfig()
			invalidConfig.Server.Name = "" // Invalid name
			
			server, err := NewAgenticMemoryServer(invalidConfig)
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(server, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid server config")
			})
		})
	})
}

func TestAgenticMemoryServerLifecycle(t *testing.T) {
	Convey("Given a valid AgenticMemoryServer", t, func() {
		config := DefaultServerConfig()
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)
		
		ctx := context.Background()
		
		Convey("When starting the server", func() {
			err := server.Start(ctx)
			
			Convey("Then it should start successfully", func() {
				So(err, ShouldBeNil)
				So(server.IsRunning(), ShouldBeTrue)
			})
			
			Convey("And when starting again", func() {
				err := server.Start(ctx)
				
				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldContainSubstring, "server is already running")
				})
			})
			
			Convey("And when stopping the server", func() {
				err := server.Stop(ctx)
				
				Convey("Then it should stop successfully", func() {
					So(err, ShouldBeNil)
					So(server.IsRunning(), ShouldBeFalse)
				})
				
				Convey("And when stopping again", func() {
					err := server.Stop(ctx)
					
					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
						So(err.Error(), ShouldContainSubstring, "server is not running")
					})
				})
			})
		})
		
		Convey("When stopping a server that was never started", func() {
			err := server.Stop(ctx)
			
			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "server is not running")
			})
		})
	})
}

func TestAgenticMemoryServerConcurrency(t *testing.T) {
	Convey("Given a valid AgenticMemoryServer", t, func() {
		config := DefaultServerConfig()
		server, err := NewAgenticMemoryServer(config)
		So(err, ShouldBeNil)
		So(server, ShouldNotBeNil)
		
		ctx := context.Background()
		
		Convey("When multiple goroutines try to start/stop concurrently", func() {
			const numGoroutines = 10
			startErrors := make(chan error, numGoroutines)
			stopErrors := make(chan error, numGoroutines)
			
			// Start multiple goroutines trying to start the server
			for i := 0; i < numGoroutines; i++ {
				go func() {
					startErrors <- server.Start(ctx)
				}()
			}
			
			// Wait a bit for starts to complete
			time.Sleep(10 * time.Millisecond)
			
			// Start multiple goroutines trying to stop the server
			for i := 0; i < numGoroutines; i++ {
				go func() {
					stopErrors <- server.Stop(ctx)
				}()
			}
			
			// Collect results
			var startSuccesses, stopSuccesses int
			for i := 0; i < numGoroutines; i++ {
				if err := <-startErrors; err == nil {
					startSuccesses++
				}
				if err := <-stopErrors; err == nil {
					stopSuccesses++
				}
			}
			
			Convey("Then only one start and one stop should succeed", func() {
				So(startSuccesses, ShouldEqual, 1)
				So(stopSuccesses, ShouldEqual, 1)
			})
		})
	})
}

// Benchmark tests
func BenchmarkNewAgenticMemoryServer(b *testing.B) {
	config := DefaultServerConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server, err := NewAgenticMemoryServer(config)
		if err != nil {
			b.Fatal(err)
		}
		_ = server
	}
}

func BenchmarkServerStartStop(b *testing.B) {
	config := DefaultServerConfig()
	server, err := NewAgenticMemoryServer(config)
	if err != nil {
		b.Fatal(err)
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := server.Start(ctx); err != nil {
			b.Fatal(err)
		}
		if err := server.Stop(ctx); err != nil {
			b.Fatal(err)
		}
	}
}