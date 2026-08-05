import { useEffect, useRef } from "react";
import * as echarts from "echarts";

interface LineChartData {
  date: string;
  income: number;
  expense: number;
  transfer?: number;
  savings?: number;
}

interface LineChartProps {
  data: LineChartData[];
  title?: string;
}

export function LineChart({ data, title }: LineChartProps): JSX.Element {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    if (!containerRef.current || !data.length) return;

    const chart = echarts.init(containerRef.current);
    chartRef.current = chart;

    const dates = data.map((d) => d.date);
    const incomeValues = data.map((d) => d.income);
    const expenseValues = data.map((d) => d.expense);
    const transferValues = data.map((d) => d.transfer || 0);
    const savingsValues = data.map((d) => d.savings || 0);

    const option = {
      tooltip: {
        trigger: "axis",
        axisPointer: { type: "cross" }
      },
      legend: {
        bottom: 0,
        left: "center",
        icon: "circle"
      },
      grid: {
        left: "3%",
        right: "4%",
        top: 24,
        bottom: 56,
        containLabel: true
      },
      xAxis: {
        type: "category",
        data: dates,
        boundaryGap: false
      },
      yAxis: {
        type: "value"
      },
      series: [
        {
          name: "Income",
          type: "line",
          data: incomeValues,
          smooth: true,
          itemStyle: { color: "#1b8f65" },
          areaStyle: { color: "rgba(27, 143, 101, 0.2)" }
        },
        {
          name: "Expense",
          type: "line",
          data: expenseValues,
          smooth: true,
          itemStyle: { color: "#c44a38" },
          areaStyle: { color: "rgba(196, 74, 56, 0.2)" }
        },
        ...(Math.max(...transferValues) > 0
          ? [
              {
                name: "Transfer",
                type: "line",
                data: transferValues,
                smooth: true,
                itemStyle: { color: "#335f9f" },
                areaStyle: { color: "rgba(51, 95, 159, 0.2)" }
              }
            ]
          : []),
        ...(Math.max(...savingsValues) > 0
          ? [
              {
                name: "Savings",
                type: "line",
                data: savingsValues,
                smooth: true,
                itemStyle: { color: "#6b4fb5" },
                areaStyle: { color: "rgba(107, 79, 181, 0.2)" }
              }
            ]
          : [])
      ]
    };

    chart.clear();
    chart.setOption(option, true);

    const handleResize = () => chart.resize();
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      chart.dispose();
    };
  }, [data, title]);

  return <div ref={containerRef} className="line-chart-surface" style={{ width: "100%", height: "320px" }} />;
}
