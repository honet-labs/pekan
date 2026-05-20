import { useEffect, useRef } from "react";
import * as echarts from "echarts";

interface PieChartProps {
  data: Array<{ name: string; value: number; color: string }>;
  onHover?: (item: { name: string; value: number } | null) => void;
}

export function PieChart({ data, onHover }: PieChartProps): JSX.Element {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    if (!containerRef.current || !data.length) return;

    const chart = echarts.init(containerRef.current);
    chartRef.current = chart;

    const option = {
      tooltip: {
        trigger: "item",
        formatter: (params: any) => `${params.name}: ${params.value}`
      },
      legend: {
        orient: "vertical",
        left: "left"
      },
      series: [
        {
          name: "Distribution",
          type: "pie",
          radius: "50%",
          data: data.map((item) => ({ value: item.value, name: item.name, itemStyle: { color: item.color } })),
          emphasis: {
            itemStyle: {
              shadowBlur: 10,
              shadowColor: "rgba(0, 0, 0, 0.5)"
            }
          }
        }
      ]
    };

    chart.setOption(option);

    const handleMouseOver = (params: any) => {
      if (params.componentSubType === "pie" && onHover) {
        const item = data.find((d) => d.name === params.name);
        if (item) {
          onHover({ name: item.name, value: item.value });
        }
      }
    };

    const handleMouseOut = () => {
      if (onHover) {
        onHover(null);
      }
    };

    chart.on("mouseover", handleMouseOver);
    chart.on("mouseout", handleMouseOut);

    const handleResize = () => chart.resize();
    window.addEventListener("resize", handleResize);

    return () => {
      window.removeEventListener("resize", handleResize);
      chart.dispose();
    };
  }, [data, onHover]);

  return <div ref={containerRef} style={{ width: "100%", height: "400px" }} />;
}
