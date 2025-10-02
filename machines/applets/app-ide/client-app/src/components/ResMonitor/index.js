import "bootstrap/dist/css/bootstrap.min.css";

import React, { Component } from "react";
import Chart from "react-google-charts";

function ResMonitor({mem, cpu}) {
  let gaugeData = [
    ["Label", "Value"],
    ["Memory", Math.trunc(mem)],
    ["CPU", Math.trunc(cpu)],
  ];
  return (
    <div className="container mt-5">
      <Chart
        width={400}
        height={"100%"}
        chartType="Gauge"
        loader={<div>Loading Chart</div>}
        data={gaugeData}
        options={{
          redFrom: 90,
          redTo: 100,
          yellowFrom: 75,
          yellowTo: 90,
          minorTicks: 5,
        }}
        rootProps={{ "data-testid": "1" }}
      />
    </div>
  );
}
export default ResMonitor;
