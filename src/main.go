package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os/exec"
	"strconv"
	"time"
)

//How many numbers are to be generated (variable discontinued)
//10000000 (ten million)
// const numberCount int = 10000000

//The highest number (exclusive) that the program will randomly generate (0 inclusive) (variable discontinued)
// const maxRandomNumber = 20
//The results of the number generation. A version is kept to prevent problems with persisting outdated webpages.
type Results struct {
	RandomNumbers       map[int64]int64
	Occurrences         map[int64]int64
	GenerationTimeStart int64
	GenerationTimeEnd   int64
	SortTimeStart       int64
	SortTimeEnd         int64
	PMean               float64
	//Population mean
	Mean float64
	//Population standard deviation
	PSD float64
	//Population variation
	PVar float64
	//Sample standard deviation
	SSD float64
	//Sample variation
	SVar float64
	//The version MUST start from 0 indicating the first number generation set. Note that the version is temporary and will be reset when the program is re-executed. (This variable is not going to be implemented functionally for some time)
	Version int
}

//Post data regarding the number generation statistics
type GeneratePOSTData struct {
	//The maximum number to be generated (exclusive; 0 inclusive)
	MaxRandom int64
	//How many times the program will generate a random number between 0 and the maximum number (exclusive)
	ProgramIterations int64
}

//Defines the results of the program after it is run.
var res Results = Results{}

//Defines the port that the server will be run on as a string.
const port string = "3000"

func generateNumsPage(w http.ResponseWriter, r *http.Request) {
	const html string = `
	<!DOCTYPE html>
	<html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Generate new batch</title>
        <style>
            @keyframes btn-shake
            {
                0% {
                    transform: rotateZ(0deg);
                }
                25% {
                    transform: rotateZ(3deg);
                }
                50% {
                    transform: rotateZ(-3deg);
                }
                100% {
                    transform: rotateZ(0deg);
                }
            }
            body
            {
                background-color: #645959;
                font-family: sans-serif;
            }
            .main-header
            {
                text-align: center;
                border-bottom: 1px solid black;
            }
            .input-information
            {
                padding-left: 10px;
                margin: 0 auto;
                width: 50%;
                border-left: 1px solid black;
            }
            .input-information input
            {
                padding: 1px;
                display: block;
                background-color: grey;
            }
            .input-information button
            {
                display: block;
                margin-top: 10px;
                border-radius: 3px;
                border: 1px solid black;
                width: 70px;
                height: 20px;
                cursor: pointer;
                background-color: grey;
                user-select: none;
            }
            .input-information-align-left
            {
                text-align: left;
            }
            #generation-status
            {
                color: white;
            }
            #generation-status:is(.gen-status-ERR)
            {
                color: darkred;
            }
        </style>
        <script>
            const load = () => {
                const btnListener = async (e) => {
                    const MAX_HIGH_NUMBER = 20000000;
                    const btn = e.srcElement;
                    btn.style.animation = "";
                    btn.style.borderColor = "";
                    btn.disabled = "true";
                    const highInput = document.querySelector("#num_high");
                    const iterInput = document.querySelector("#prgm_iter");
                    const genStatusP = document.querySelector("#generation-status");
                    iterInput.style.borderColor = "";
                    highInput.style.borderColor = "";
                    genStatusP.classList.remove("gen-status-ERR");
                    genStatusP.textContent = "";
                    const high = Number(document.querySelector("#num_high").value);
                    let iter = Number(document.querySelector("#prgm_iter").value);
                    if(iter > MAX_HIGH_NUMBER) {
                        btn.style.animation = "btn-shake 0.2s linear";
                        btn.style.borderColor = "red";
                        iterInput.style.borderColor = "red";
                        btn.disabled = "";
                    }
                    if(high < 1) {
                        btn.style.animation = "btn-shake 0.2s linear";
                        btn.style.borderColor = "red";
                        highInput.style.borderColor = "red";
                        btn.disabled = "";   
                    }
                    if (iter === '') iter = 0;
                    //Added it after so that both can be red.
                    if(high < 1 || iter > MAX_HIGH_NUMBER) return;
                    let res = {
                        ok: false,
                        status: "N/A"
                    };
                    let rJSON = {};
                    try{
                        res = await fetch('/generateNums',  {
                            method: "POST",
                            body: JSON.stringify({
                                "MaxRandom": high,
                                "ProgramIterations": iter
                            })
                        });
                        rJSON = await res.json();
                    }
                    catch(err){

                    }
                    if(res.ok){
                        if(genStatusP.classList.contains("gen-status-ERR")){
                            genStatusP.classList.toggle("gen-status-ERR");
                        }
                        genStatusP.textContent = "Generation finished. Proceed to http://localhost:" + rJSON.Port + " to see results.";
                    }
                    else{
                        if(!genStatusP.classList.contains("gen-status-ERR")){
                            genStatusP.classList.toggle("gen-status-ERR");
                        }
                        genStatusP.textContent = "Generation failed with status " + res.status + "! Try again...";
                    }
                    btn.disabled = "";
                };
                document.querySelector("#generate-btn").addEventListener("click", btnListener);
            };
            window.addEventListener("load", load);
        </script>
    </head>
    <body>
        <main>
            <header class="main-header">
                <h1>Generate numbers</h1>
				<p>Notice: results may be inaccurate past 10 decimal places. However, these inaccuracies should be too miniscule to be problematic in rounded statistical data.</p>
            </header>
            <section class="input-information">
                <div class="input-information-align-left"></div>
                <h3>Input program parameters</h3>
                <p>Number high (exclusive) - number low will be 0</p>
                <h6>Number must be at least 1 or higher</h6>
                <input type="number" id = "num_high">
                <p>Program iterations (max is 20 million)</p>
                <input type="number" id="prgm_iter">
                <button id="generate-btn">Submit</button>
                <p id="generation-status" class="success"></p>
            </section>
        </main>
    </body>
	</html>
	`
	w.Write([]byte(html))
}

func generateNums(w http.ResponseWriter, r *http.Request) {
	var POSTDat = GeneratePOSTData{}
	decodeErr := json.NewDecoder(r.Body).Decode(&POSTDat)
	if decodeErr != nil {
		http.Error(w, "Cannot generate numbers - error decoding POST data", http.StatusInternalServerError)
		log.Fatal(decodeErr)
		return
	}
	fmt.Println("Generating new batch...\nSettings:")
	fmt.Println("   generator_iterations=" + fmt.Sprint(POSTDat.ProgramIterations))
	fmt.Println("   max_rand_num=" + fmt.Sprint(POSTDat.MaxRandom))
	//Starting time for the random number generation
	var generationTimeStart time.Time = time.Now()
	var randoms = make(map[int64]int64)
	var i int64 = 0
	for i < POSTDat.ProgramIterations {
		rand.Seed(time.Now().UnixMicro())
		randoms[i] = int64(rand.Int63n(POSTDat.MaxRandom))
		i++
	}
	//Ending time for the random number generation
	var generationTimeEnd time.Time = time.Now()
	//Starting time for the number sorting
	var sortTimeStart time.Time = time.Now()
	var sort = make(map[int64]int64)
	for _, val := range randoms {
		sort[val] = 0
	}
	for _, val := range randoms {
		sort[val] += 1
	}
	//Ending time for the number sorting
	var sortTimeEnd time.Time = time.Now()
	//Mean & Standard Deviation calculation time
	// var test = make(map[int64]int64)
	// test[0] = 10
	// test[1] = 3
	// test[2] = 4
	// test[3] = 16
	// test[4] = 12
	// test[5] = 5
	// test[6] = 2
	// test[7] = 7
	// test[8] = 8
	// test[9] = 7

	var mean float64 = 0
	var meanCounter int64 = 0
	//Sum of the frequencies
	for meanCounter < int64(len(randoms)) {
		mean += float64(randoms[meanCounter])
		meanCounter++
	}
	mean /= float64(len(randoms))
	//Oh boy, here comes the standard deviation calculation...
	var SD float64 = 0
	//Summation of the data points minus the mean data point (subtraction operation comes first), squared
	var summation float64 = 0
	var SDCounter int64 = 0
	for SDCounter < int64(len(randoms)) {
		summation += math.Pow(float64(randoms[SDCounter])-mean, 2)
		SDCounter++
	}
	summation /= float64(len(randoms) - 1)
	summation = math.Sqrt(summation)
	SD = summation
	var PMean float64 = 0
	var PMeanCounter int64 = 0
	for PMeanCounter < POSTDat.MaxRandom {
		PMean += float64(PMeanCounter) * float64(float64(1)/float64(POSTDat.MaxRandom))
		PMeanCounter++
	}
	var PSD float64 = 0
	var PSDCounter int64 = 0
	for PSDCounter < POSTDat.MaxRandom {
		PSD += math.Pow(float64(PSDCounter), 2) * float64(float64(1)/float64(POSTDat.MaxRandom))
		PSDCounter++
	}
	PSD -= math.Pow(PMean, 2)
	PSD = math.Sqrt(PSD)
	res = Results{
		randoms,
		sort,
		generationTimeStart.UnixMicro(),
		generationTimeEnd.UnixMicro(),
		sortTimeStart.UnixMicro(),
		sortTimeEnd.UnixMicro(),
		PMean,
		mean,
		PSD,
		math.Pow(PSD, 2),
		SD,
		math.Pow(SD, 2),
		0,
	}
	fmt.Println("Mean: "+strconv.FormatFloat(mean, 'f', -1, 64), "Standard deviation: "+strconv.FormatFloat(SD, 'f', -1, 64), "Population mean: "+strconv.FormatFloat(PMean, 'f', -1, 64), "Population standard deviation: "+strconv.FormatFloat(PSD, 'f', -1, 64), "Sample variance: "+strconv.FormatFloat(math.Pow(SD, 2), 'f', -1, 64), "Population variance: "+strconv.FormatFloat(math.Pow(SD, 2), 'f', -1, 64))
	var returnRes = struct {
		Port string
	}{
		port,
	}
	returnResJSON, err := json.Marshal(returnRes)
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Cannot handle request - internal server error", http.StatusInternalServerError)
	}
	fmt.Println("Done!")
	w.Write([]byte(returnResJSON))
}

func results(w http.ResponseWriter, r *http.Request) {
	var html string = `
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>Calculation results</title>
			<style>
				body
				{
					background-color: #645959;
					font-family: sans-serif;
				}
				.main-header
				{
					padding-bottom: 10px;
					padding-left: 50px;
					border-bottom: 1px solid black;
					color: #ffeeee;
				}
				.divider_1px
				{
					width: 50%;
					border-bottom: 1px solid black;
				}
				.divider_2px
				{
					width: 50%;
					border-bottom: 2px solid black;
				}
				.information-section
				{
					border-bottom: 1px solid black;
				}
				.time
				{
					padding-left: 10px;
				}
				th, td
				{
					border: 1px solid black;
				}
			</style>
			<script>
				const {DOMParser} = globalThis;
				const onload = async () => {
					const occurrences = await (await fetch("/getOccurrences")).json();
					document.querySelector("#tr_loading").remove();
					Object.entries(occurrences).forEach(obj => {
						const num = obj[0];
						const occ = obj[1];
						//Cannot use templating because otherwise it ends the string in Go (which cannot be escaped), causing errors
						const tr = document.createElement("tr");
						tr.innerHTML = "<td>" + num + "</td><td>" + occ + "</td>";
						document.querySelector("#frequency-distribution").appendChild(tr);
					});
				};
				window.addEventListener("load", onload);
			</script>
		</head>
		<body>
			<header class="main-header">
				<h1>Test results</h1>
				<p>Notice: results may be inaccurate past 10 decimal places. However, these inaccuracies should be too miniscule to be problematic in rounded statistical data.</p> 
			</header>
			<main>
				<section class="time information-section">
					<h1>Time</h1>
					<p>Total time taken for number generation: ~` + strconv.Itoa(int((res.GenerationTimeEnd-res.GenerationTimeStart)/1000000)) + ` seconds (start in Unix micro: ` + strconv.Itoa(int(res.GenerationTimeStart)) + `; end in Unix micro: ` + strconv.Itoa(int(res.GenerationTimeEnd)) + `)</p>
					<p>Total time taken for result classification and processing: ~` + strconv.Itoa(int((res.SortTimeEnd-res.SortTimeStart)/1000000)) + ` seconds (start in Unix micro: ` + strconv.Itoa(int(res.SortTimeStart)) + `; end in Unix micro: ` + strconv.Itoa(int(res.SortTimeEnd)) + `)</p>
					<div class="divider_2px"></div>
					<h3>Total time taken to complete: ~` + strconv.Itoa(int((res.SortTimeEnd-res.GenerationTimeStart)/1000000)) + ` seconds (start in Unix micro: ` + strconv.Itoa(int(res.GenerationTimeStart)) + `; end in Unix micro: ` + strconv.Itoa(int(res.SortTimeEnd)) + `</h3>
				</section>
				<section class="summary-statistics information-section">
                	<h1>Summary Statistics</h1>
                	<p>Mean: ` + strconv.FormatFloat(res.Mean, 'f', -1, 64) + `</p>
                	<p>Standard deviation: ` + strconv.FormatFloat(res.SSD, 'f', -1, 64) + `</p>
                	<p>Population mean: ` + strconv.FormatFloat(res.PMean, 'f', -1, 64) + `</p>
                	<p>Population standard deviation: ` + strconv.FormatFloat(res.PSD, 'f', -1, 64) + `</p>
					<p>Sample Variance: ` + strconv.FormatFloat(res.SVar, 'f', -1, 64) + `</p>
					<p>Population Variance: ` + strconv.FormatFloat(res.PVar, 'f', -1, 64) + `</p>
            	</section>
				<section class="frequency-table information-section">
					<h1>Frequency Distribution of Occurrences</h1>
					<table id="frequency-distribution">
						<tr>
							<th>Number</th>
							<th>Occurrences of that number</th>
						</tr>
						<tr id="tr_loading">
							<td>Loading...</td>
							<td>Loading...</td>
						</tr>
					</table>
				</section>
			</main>
		</body>
	</html>
	`
	w.Write([]byte(html))
}

func returnOccurrences(w http.ResponseWriter, r *http.Request) {
	j, err := json.Marshal(res.Occurrences)
	if err != nil {
		http.Error(w, "Cannot get occurrences - internal server error", http.StatusInternalServerError)
		return
	}
	w.Write([]byte(j))
}

func openBrowser(url string) bool {
	// var args []string
	// args = []string{"/c", "start"}
	cmd := exec.Command("cmd", "/c", "start", url)
	return cmd.Start() == nil
}

func main() {
	// fmt.Println("Timing:")
	// fmt.Println("   time_unit=uix_micro")
	// fmt.Println("   generation_time_start=" + fmt.Sprint(generationTimeStart.UnixMicro()))
	// fmt.Println("   generation_time_end=" + fmt.Sprint(generationTimeEnd.UnixMicro()))
	// fmt.Println("   generation_time_total=" + fmt.Sprint(generationTimeEnd.UnixMicro()-generationTimeStart.UnixMicro()))
	// fmt.Println("   sort_time_start=" + fmt.Sprint(sortTimeStart.UnixMicro()))
	// fmt.Println("   sort_time_end=" + fmt.Sprint(sortTimeEnd.UnixMicro()))
	// fmt.Println("   sort_time_total=" + fmt.Sprint(sortTimeEnd.UnixMicro()-sortTimeStart.UnixMicro()))
	// fmt.Println("Program execution was successful!")
	// fmt.Println("Processing data and creating local webpage for visuals...")
	// fmt.Println("done!")
	// fmt.Println("Configuring web server...")
	res.Version = 0
	http.HandleFunc("/", results)
	http.HandleFunc("/generate", generateNumsPage)
	http.HandleFunc("/generateNums", generateNums)
	http.HandleFunc("/getOccurrences", returnOccurrences)
	fmt.Println("done!")
	fmt.Println("Starting web server. Navigate to localhost:" + port + " to see results")
	log.Println("Listening on port :" + port)
	openBrowser("http://localhost:" + port + "/generate")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
