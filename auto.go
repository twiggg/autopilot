package main

import (
	"fmt"
	//"sync"
	"math"
	"time"
)

//----------------------- PROGRAM ----------------------------------

func main() {
	fmt.Println("started...")
	defer fmt.Println("end of program")

	//define the vehicule properties : name, weight in kg, start position, max speed in km/h and 0to100km/h in seconds
	carA := NewCar("car A", 100, Position{0, 0, 0}, 180, 2)
	//show caracteristics of the defined vehicule
	carA.Caract()

	//define the trip
	//add x and y coordinates of the steps
	carA.AddDest(0, 1500)    //add a first destination to the trip
	carA.AddDest(2100, -300) //add a second destination (the trip is made in 2 steps)
	//initiate GPS
	carA.Start() //sets departure
	//compute trajectory for the first step of the trip
	carA.Orientation()

	//this is a timeout, that sends a signal if vehicule has not arrived in time
	timeout := 10
	go func() {
		<-time.After(time.Duration(timeout*1000) * time.Millisecond)
		fmt.Printf("TIMEOUT: %.3fs passed by\n", float64(timeout))
		carA.stopper <- true
		//fmt.Println("sent stop order")
	}()

	//proceed to trip
	ping1 := 50 //means that the computation is updated every 'ping1' milliseconds.
	ping2 := 500 //means that the route info will be printed every 'ping2' milliseconds.
	carA.Move(ping1,ping2)

}

// ----------------------- DEFINITIONS -------------------------------

type Mover interface {
	Move()
	//Position() Position
	//StartAt(x,y int)
	//Destination(x,y int)
	//Stop()
}

type Car struct {
	name        string
	weight      int
	prevpos     Position //previous position
	pos         Position
	dest        []Position
	to          Position
	distance    float64 //total distance
	speed       float64 //current speed
	maxspeed    float64 //in m/ms
	zToH        float64 //in ms
	traj        Trajectory
	stopped     bool
	stopper     chan bool
	portion     int
	angle1      float64
	orientation string
	dx          float64
	dy          float64
}

//set the max speed in km/h and the zero to hundred time in s
func NewCar(n string, w int, start Position, maxS float64, zerotohund float64) *Car {
	channel := make(chan bool, 1)
	car := Car{name: n, weight: w, pos: start, prevpos: start, maxspeed: kmphTompms(maxS), zToH: zerotohund * 1000, stopped: false, stopper: channel}
	return &car
}
func (c *Car) SetStart(x float64, y float64) {
	c.pos.x = x
	c.pos.y = y
	c.prevpos.x = x
	c.prevpos.y = y
}
func (c *Car) AddDest(x float64, y float64) {
	c.dest = append(c.dest, Position{x: x, y: y})
	/*c.dest.x = x
	c.dest.y = y*/
}

func (c *Car) Caract() {
	fmt.Printf("\nname:%s\nmax speed:%dkm/h\n0 to 100km/h in %ds\n\n", c.name, int(mpmsTokmph(c.maxspeed)), int(c.zToH/1000))
}

func (c *Car) Move(ping1,ping2 int) {
	//ping := 80 //ms
	t := 0
	i := 1
	if c.portion == 0 {
		c.NewPos(0)
	}
	for {
		select {
		case stop := <-c.stopper:
			fmt.Println("received stop order!")
			c.stopped = stop
			return
		case <-time.After(time.Duration(ping1) * time.Millisecond):
			//fmt.Println(<-c.stopper)
			t = ping1 * i
			c.Speed(int64(t))
			//update speed to compute next position
			msg:=c.NewPos(int64(t))
			if t%ping2==0{fmt.Printf("%s",msg)}
			i++
		}

	}
}

func (c *Car) Start() {
	c.portion = 0
}
func (c *Car) Orientation() {
	if c.portion == 0 {
		if len(c.dest) == 0 {
			fmt.Println("no trip")
			return
		}
		if len(c.dest) == 1 {
			fmt.Printf("\none step trip from (x:%f ; y:%f) to (x:%f ; y:%f)\n", c.pos.x, c.pos.y, c.dest[0].x, c.dest[0].y)
		} else {
			fmt.Printf("\nSTEP %d from (x:%f ; y:%f) to (x:%f ; y:%f)\n", c.portion+1,c.pos.x, c.pos.y, c.dest[0].x, c.dest[0].y)
		}
	} else {
		fmt.Printf("\nSTEP %d from (x:%f ; y:%f) to (x:%f ; y:%f)\n", c.portion+1, c.pos.x, c.pos.y, c.dest[c.portion].x, c.dest[c.portion].y)
	}

	//compute angle ---------------------------------------
	c.dy = c.dest[c.portion].y - c.pos.y
	c.dx = c.dest[c.portion].x - c.pos.x
	if c.dx == 0 {
		if c.dy > 0 {
			c.angle1 = 90
		} else {
			c.angle1 = -90
		}
	} else if c.dy == 0 {
		if c.dx > 0 {
			c.angle1 = 0
		} else {
			c.angle1 = 180
		}
	} else {
		c.angle1 = math.Atan(c.dy/c.dx) * 180 / math.Pi
	}
	// compute orientation -------------------------------------
	c.angle1 = corrangle(c.angle1, c.dy, c.dx)
	c.orientation = oriVal(c.angle1)
}

//time in ms
func (c *Car) NewPos(deltat int64) string{
	if len(c.dest) == 0 {
		c.stopper <- true
		return fmt.Sprintf("\nno trip planned yet, please add a destination\n")

	}
	if c.pos.x == c.dest[c.portion].x && c.pos.y == c.dest[c.portion].y {
		if c.portion == len(c.dest)-1 {
			c.stopper <- true
			fmt.Printf("\narrived to destination in %.2f seconds\n", float64(c.pos.t)/float64(1000))
		} else {
			c.portion++
			//c.start=Position{x:c.pos.x,y:c.pos.y,t:deltat}
			//c.UpdateTraj()
			c.Orientation()
			//fmt.Println("next step of the trip")
		}
	}
	//check direction (sign of the vector)
	signx := c.dest[c.portion].x - c.prevpos.x
	signy := c.dest[c.portion].y - c.prevpos.y
	//step
	c.prevpos.x = c.pos.x
	c.prevpos.y = c.pos.y
	c.prevpos.t = c.pos.t
	c.pos.t = deltat
	//check how long this process takes and make sure that the ping time in not shorter that this

	//precompute next position
	//nx := c.prevpos.x + math.Cos(math.Atan(c.traj.slope))*c.speed*float64(deltat)
	//ny := c.prevpos.y + math.Sin(math.Atan(c.traj.slope))*c.speed*float64(deltat)
	nx := c.prevpos.x + math.Cos(c.angle1)*c.speed*float64(deltat)
	ny := c.prevpos.y + math.Sin(c.angle1)*c.speed*float64(deltat)
	//check if next move is the last
	//check x
	switch signx > 0 {
	case true:
		if nx > c.dest[c.portion].x {
			c.pos.x = c.dest[c.portion].x
		} else {
			c.pos.x = nx
		}
	case false:
		if nx < c.dest[c.portion].x {
			c.pos.x = c.dest[c.portion].x
		} else {
			c.pos.x = nx
		}
	}
	//check y
	switch signy > 0 {
	case true:
		if ny > c.dest[c.portion].y {
			c.pos.y = c.dest[c.portion].y
		} else {
			c.pos.y = ny
		}
	case false:
		if ny < c.dest[c.portion].y {
			c.pos.y = c.dest[c.portion].y
		} else {
			c.pos.y = ny
		}
	}

	return fmt.Sprintf("t=%.3fs --> x: %f y:%f , speed:%.1fkm/h, orientation:%s\n", float64(c.pos.t)/float64(1000), c.pos.x, c.pos.y, mpmsTokmph(c.speed),c.orientation)

}

//update and return the speed in m/ms
func (c *Car) Speed(deltat int64) float64 {
	pente := float64(kmphTompms(100) / c.zToH)
	nspeed := pente * float64(deltat)
	if nspeed <= c.maxspeed {
		c.speed = nspeed
	} else {
		c.speed = c.maxspeed
	}
	return c.speed
}

type Trajectory struct {
	slope float64
	zero  float64
}

type Position struct {
	x float64
	y float64
	t int64 //time.Time
}

func kmphTompms(kmh float64) float64 {
	//takes a speed in km/h and converse it to m/ms
	return kmh / 3600
}

func mpmsTokmph(mpms float64) float64 {
	//takes a speed in km/h and converse it to m/ms
	return mpms * 3600
}

func corrangle(angle, dy, dx float64) float64 {
	if dy == 0 && dx > 0 {
		return 0
	}
	if dy == 0 && dx < 0 {
		return 180
	}
	if dx == 0 && dy > 0 {
		return 90
	}
	if dx == 0 && dy < 0 {
		return -90
	}
	if dx > 0 && dy > 0 {
		return angle
	}
	if dx > 0 && dy < 0 {
		return 360 + angle
	}
	if dx < 0 && dy > 0 {
		return 180 - angle
	}
	if dx < 0 && dy < 0 {
		return 180 + 90 - angle
	}
	return angle
}
func oriVal(angle float64) string {
	ori := []string{"X", "XY", "Y", "-XY", "-X", "-X-Y", "-Y", "X-Y"}
	/*ind:=int((angle-360/8/2)/(360/8))
	if (angle)
	fmt.Println("ind:",ind)
	if ind<len(ori)-1{
	return ori[ind]
	}
	return ori[0]
	*/

	for i := 0; i < 8; i++ {
		ainf := float64(-22.5 + float64(i)*45)
		asup := float64(22.5 + float64(i)*45)
		if angle >= ainf && angle <= asup {
			return ori[i]
		}
	}
	return ori[0]

}
