// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package static

const containersJs = `
google.load("visualization", "1", {packages: ["corechart", "gauge"]});

// Draw a line chart.
function drawLineChart(seriesTitles, data, elementId, unit) {
	var min = Infinity;
	var max = -Infinity;
	for (var i = 0; i < data.length; i++) {
		// Convert the first column to a Date.
		if (data[i] != null) {
			data[i][0] = new Date(data[i][0]);
		}

		// Find min, max.
		for (var j = 1; j < data[i].length; j++) {
			var val = data[i][j];
			if (val < min) {
				min = val;
			}
			if (val > max) {
				max = val;
			}
		}
	}

	// We don't want to show any values less than 0 so cap the min value at that.
	// At the same time, show 10% of the graph below the min value if we can.
	var minWindow = min - (max - min) / 10;
	if (minWindow < 0) {
		minWindow = 0;
	}

	// Add the definition of each column and the necessary data.
	var dataTable = new google.visualization.DataTable();
	dataTable.addColumn('datetime', seriesTitles[0]);
	for (var i = 1; i < seriesTitles.length; i++) {
		dataTable.addColumn('number', seriesTitles[i]);
	}
	dataTable.addRows(data);

	// Create and draw the visualization.
	if (!(elementId in window.charts)) {
		window.charts[elementId] = new google.visualization.LineChart(document.getElementById(elementId));
	}

	// TODO(vmarmol): Look into changing the view window to get a smoother animation.
	var opts = {
		curveType: 'function',
		height: 300,
		legend:{position:"none"},
		focusTarget: "category",
		vAxis: {
			title: unit,
			viewWindow: {
				min: minWindow,
			},
		},
		legend: {
			position: 'bottom',
		},
	};

	window.charts[elementId].draw(dataTable, opts);
}

// Gets the length of the interval in nanoseconds.
function getInterval(current, previous) {
	var cur = new Date(current);
	var prev = new Date(previous);

	// ms -> ns.
	return (cur.getTime() - prev.getTime()) * 1000000;
}

// Checks if the specified stats include the specified resource.
function hasResource(stats, resource) {
	return stats.stats.length > 0 && stats.stats[0][resource];
}

// Draw a gauge.
function drawGauge(elementId, cpuUsage, memoryUsage) {
	var gauges = [['Label', 'Value']];
	if (cpuUsage >= 0) {
		gauges.push(['CPU', cpuUsage]);
	}
	if (memoryUsage >= 0) {
		gauges.push(['Memory', memoryUsage]);
	}
	// Create and populate the data table.
	var data = google.visualization.arrayToDataTable(gauges);

	// Create and draw the visualization.
	var options = {
		width: 400, height: 120,
		redFrom: 90, redTo: 100,
		yellowFrom:75, yellowTo: 90,
		minorTicks: 5,
		animation: {
			duration: 900,
			easing: 'linear'
		}
	};
	var chart = new google.visualization.Gauge(document.getElementById(elementId));
	chart.draw(data, options);
}

// Get the machine info.
function getMachineInfo(callback) {
	$.getJSON("/api/v1.0/machine", function(data) {
		callback(data);
	});
}

// Get the container stats for the specified container.
function getStats(containerName, callback) {
	// Request 60s of container history and no samples.
	var request = JSON.stringify({
                // Update main.statsRequestedByUI while updating "num_stats" here.
		"num_stats": 60,
		"num_samples": 0
	});
	$.post("/api/v1.0/containers" + containerName, request, function(data) {
		callback(data);
	}, "json");
}

// Draw the graph for CPU usage.
function drawCpuTotalUsage(elementId, machineInfo, stats) {
	if (stats.spec.has_cpu && !hasResource(stats, "cpu")) {
		return;
	}

	var titles = ["Time", "Total"];
	var data = [];
	for (var i = 1; i < stats.stats.length; i++) {
		var cur = stats.stats[i];
		var prev = stats.stats[i - 1];
		var intervalInNs = getInterval(cur.timestamp, prev.timestamp);

		var elements = [];
		elements.push(cur.timestamp);
		elements.push((cur.cpu.usage.total - prev.cpu.usage.total) / intervalInNs);
		data.push(elements);
	}
	drawLineChart(titles, data, elementId, "Cores");
}

// Draw the graph for per-core CPU usage.
function drawCpuPerCoreUsage(elementId, machineInfo, stats) {
	if (stats.spec.has_cpu && !hasResource(stats, "cpu")) {
		return;
	}

	// Add a title for each core.
	var titles = ["Time"];
	for (var i = 0; i < machineInfo.num_cores; i++) {
		titles.push("Core " + i);
	}
	var data = [];
	for (var i = 1; i < stats.stats.length; i++) {
		var cur = stats.stats[i];
		var prev = stats.stats[i - 1];
		var intervalInNs = getInterval(cur.timestamp, prev.timestamp);

		var elements = [];
		elements.push(cur.timestamp);
		for (var j = 0; j < machineInfo.num_cores; j++) {
			elements.push((cur.cpu.usage.per_cpu_usage[j] - prev.cpu.usage.per_cpu_usage[j]) / intervalInNs);
		}
		data.push(elements);
	}
	drawLineChart(titles, data, elementId, "Cores");
}

// Draw the graph for CPU usage breakdown.
function drawCpuUsageBreakdown(elementId, containerInfo) {
	if (containerInfo.spec.has_cpu && !hasResource(containerInfo, "cpu")) {
		return;
	}

	var titles = ["Time", "User", "Kernel"];
	var data = [];
	for (var i = 1; i < containerInfo.stats.length; i++) {
		var cur = containerInfo.stats[i];
		var prev = containerInfo.stats[i - 1];
		var intervalInNs = getInterval(cur.timestamp, prev.timestamp);

		var elements = [];
		elements.push(cur.timestamp);
		elements.push((cur.cpu.usage.user - prev.cpu.usage.user) / intervalInNs);
		elements.push((cur.cpu.usage.system - prev.cpu.usage.system) / intervalInNs);
		data.push(elements);
	}
	drawLineChart(titles, data, elementId, "Cores");
}

// Draw the gauges for overall resource usage.
function drawOverallUsage(elementId, machineInfo, containerInfo) {
	var cur = containerInfo.stats[containerInfo.stats.length - 1];

	var cpuUsage = 0;
	if (containerInfo.spec.has_cpu && containerInfo.stats.length >= 2) {
		var prev = containerInfo.stats[containerInfo.stats.length - 2];
		var rawUsage = cur.cpu.usage.total - prev.cpu.usage.total;
		var intervalInNs = getInterval(cur.timestamp, prev.timestamp);

		// Convert to millicores and take the percentage
		cpuUsage = Math.round(((rawUsage / intervalInNs) / machineInfo.num_cores) * 100);
		if (cpuUsage > 100) {
			cpuUsage = 100;
		}
	}

	var memoryUsage = 0;
	if (containerInfo.spec.has_memory) {
		// Saturate to the machine size.
		var limit = containerInfo.spec.memory.limit;
		if (limit > machineInfo.memory_capacity) {
			limit = machineInfo.memory_capacity;
		}

		memoryUsage = Math.round((cur.memory.usage / limit) * 100);
	}

	drawGauge(elementId, cpuUsage, memoryUsage);
}

var oneMegabyte = 1024 * 1024;

function drawMemoryUsage(elementId, containerInfo) {
	if (containerInfo.spec.has_memory && !hasResource(containerInfo, "memory")) {
		return;
	}

	var titles = ["Time", "Total", "Hot"];
	var data = [];
	for (var i = 0; i < containerInfo.stats.length; i++) {
		var cur = containerInfo.stats[i];

		var elements = [];
		elements.push(cur.timestamp);
		elements.push(cur.memory.usage / oneMegabyte);
		elements.push(cur.memory.working_set / oneMegabyte);
		data.push(elements);
	}
	drawLineChart(titles, data, elementId, "Megabytes");
}

// Draw the graph for network tx/rx bytes.
function drawNetworkBytes(elementId, machineInfo, stats) {
	if (stats.spec.has_network && !hasResource(stats, "network")) {
		return;
	}

	var titles = ["Time", "Tx bytes", "Rx bytes"];
	var data = [];
	for (var i = 1; i < stats.stats.length; i++) {
		var cur = stats.stats[i];
		var prev = stats.stats[i - 1];
		var intervalInSec = getInterval(cur.timestamp, prev.timestamp) / 1000000000;

		var elements = [];
		elements.push(cur.timestamp);
		elements.push((cur.network.tx_bytes - prev.network.tx_bytes) / intervalInSec);
		elements.push((cur.network.rx_bytes - prev.network.rx_bytes) / intervalInSec);
		data.push(elements);
	}
	drawLineChart(titles, data, elementId, "Bytes per second");
}

// Draw the graph for network errors
function drawNetworkErrors(elementId, machineInfo, stats) {
	if (stats.spec.has_network && !hasResource(stats, "network")) {
		return;
	}

	var titles = ["Time", "Tx", "Rx"];
	var data = [];
	for (var i = 1; i < stats.stats.length; i++) {
		var cur = stats.stats[i];
		var prev = stats.stats[i - 1];
		var intervalInSec = getInterval(cur.timestamp, prev.timestamp) / 1000000000;

		var elements = [];
		elements.push(cur.timestamp);
		elements.push((cur.network.tx_errors - prev.network.tx_errors) / intervalInSec);
		elements.push((cur.network.rx_errors - prev.network.rx_errors) / intervalInSec);
		data.push(elements);
	}
	drawLineChart(titles, data, elementId, "Errors per second");
}

// Expects an array of closures to call. After each execution the JS runtime is given control back before continuing.
// This function returns asynchronously
function stepExecute(steps) {
	// No steps, stop.
	if (steps.length == 0) {
		return;
	}

	// Get a step and execute it.
	var step = steps.shift();
	step();

	// Schedule the next step.
	setTimeout(function() {
		stepExecute(steps);
	}, 0);
}

// Draw all the charts on the page.
function drawCharts(machineInfo, containerInfo) {
	var steps = [];

	if (containerInfo.spec.has_cpu || containerInfo.spec.has_memory) {
		steps.push(function() {
			drawOverallUsage("usage-gauge", machineInfo, containerInfo)
		});
	}

	// CPU.
	if (containerInfo.spec.has_cpu) {
		steps.push(function() {
			drawCpuTotalUsage("cpu-total-usage-chart", machineInfo, containerInfo);
		});
		steps.push(function() {
			drawCpuPerCoreUsage("cpu-per-core-usage-chart", machineInfo, containerInfo);
		});
		steps.push(function() {
			drawCpuUsageBreakdown("cpu-usage-breakdown-chart", containerInfo);
		});
	}

	// Memory.
	if (containerInfo.spec.has_memory) {
		steps.push(function() {
			drawMemoryUsage("memory-usage-chart", containerInfo);
		});
	}

	// Network.
	if (containerInfo.spec.has_network) {
		steps.push(function() {
			drawNetworkBytes("network-bytes-chart", machineInfo, containerInfo);
		});
		steps.push(function() {
			drawNetworkErrors("network-errors-chart", machineInfo, containerInfo);
		});
	}

	stepExecute(steps);
}

// Executed when the page finishes loading.
function startPage(containerName, hasCpu, hasMemory) {
	// Don't fetch data if we don't have any resource.
	if (!hasCpu && !hasMemory) {
		return;
	}

	window.charts = {};

	// Get machine info, then get the stats every 1s.
	getMachineInfo(function(machineInfo) {
		setInterval(function() {
			getStats(containerName, function(stats){
				drawCharts(machineInfo, stats);
			});
		}, 1000);
	});
}
`
