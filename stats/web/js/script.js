import ApexCharts from 'apexcharts';
import {
  easepick,
  RangePlugin,
  PresetPlugin,
  TimePlugin,
} from '@easepick/bundle';

function getDateRanges() {
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);

  const last7DaysStart = new Date(today);
  last7DaysStart.setDate(today.getDate() - 6);
  const last7DaysEnd = new Date();

  const last14DaysStart = new Date(today);
  last14DaysStart.setDate(today.getDate() - 13);
  const last14DaysEnd = new Date();

  const last30DaysStart = new Date(today);
  last30DaysStart.setDate(today.getDate() - 29);
  const last30DaysEnd = new Date();

  const last90DaysStart = new Date(today);
  last90DaysStart.setDate(today.getDate() - 89);
  const last90DaysEnd = new Date();

  const last180DaysStart = new Date(today);
  last180DaysStart.setDate(today.getDate() - 179);
  const last180DaysEnd = new Date();

  const thisMonthStart = new Date(today.getFullYear(), today.getMonth(), 1);
  const thisMonthEnd = new Date(today.getFullYear(), today.getMonth() + 1, 0);

  const lastMonthStart = new Date(today.getFullYear(), today.getMonth() - 1, 1);
  const lastMonthEnd = new Date(today.getFullYear(), today.getMonth(), 0);

  const thisYearStart = new Date(today.getFullYear(), 0, 1);
  const thisYearEnd = new Date(today.getFullYear(), 11, 31);

  const lastYearStart = new Date(today.getFullYear() - 1, 0, 1);
  const lastYearEnd = new Date(today.getFullYear() - 1, 11, 31);

  const allTimeStart = new Date(1971, 0, 1);
  const allTimeEnd = new Date();

  return {
    Today: [today, today],
    Yesterday: [yesterday, yesterday],
    'Last 7 days': [last7DaysStart, last7DaysEnd],
    'Last 14 days': [last14DaysStart, last14DaysEnd],
    'Last 30 days': [last30DaysStart, last30DaysEnd],
    'Last 90 days': [last90DaysStart, last90DaysEnd],
    'Last 180 days': [last180DaysStart, last180DaysEnd],
    'This month': [thisMonthStart, thisMonthEnd],
    'Last month': [lastMonthStart, lastMonthEnd],
    'This year': [thisYearStart, thisYearEnd],
    'Last year': [lastYearStart, lastYearEnd],
    Everything: [allTimeStart, allTimeEnd],
  };
}

function toTitleCase(str) {
  return str.replace(/\w\S*/g, function (word) {
    return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
  });
}

function formatDate(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');

  return `${year}-${month}-${day}`;
}

function toHoursAndMinutes(totalMinutes) {
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;

  return `${hours}h${minutes > 0 ? ` ${minutes}m` : ''}`;
}

function plotSummary(data) {
  document.querySelector('#js-total-time').textContent = toHoursAndMinutes(
    Math.floor(data.totals.duration / 60000000000)
  );
  document.querySelector(
    '#js-top-tag-text'
  ).textContent = `${data.tags[0].name}`;
  document.querySelector(
    '#js-top-tag-hours'
  ).textContent = `(${toHoursAndMinutes(
    Math.floor(data.tags[0].duration / 60000000000)
  )})`;
  document.querySelector('#js-completed').textContent = data.totals.completed;
  document.querySelector('#js-abandoned').textContent = data.totals.abandoned;
}

function getChartOptions(seriesData, xaxisCategories, title) {
  const seriesName = 'Focus time';
  const tooltip = {
    y: {
      formatter: (value) => {
        return toHoursAndMinutes(value);
      },
    },
  };

  return {
    series: [
      {
        name: seriesName,
        data: seriesData,
      },
    ],
    chart: {
      type: 'bar',
      toolbar: {
        show: false,
      },
      height: 300,
    },
    dataLabels: {
      enabled: false,
    },

    tooltip: tooltip,
    yaxis: {
      title: {
        text: 'minutes',
      },
    },
    xaxis: {
      categories: xaxisCategories,
    },
    title: {
      text: title,
      margin: 20,
      style: {
        fontSize: '24px',
      },
    },
  };
}

function plotWeekday(data) {
  const weekdayData = [];
  const weekCategories = [];

  data.weekday.forEach((item) => {
    weekdayData.push(Math.floor(item.duration / 60000000000));
    weekCategories.push(item.name);
  });

  const weekdayOptions = getChartOptions(
    weekdayData,
    weekCategories,
    'Weekday totals'
  );

  const weekdayChart = new ApexCharts(
    document.querySelector('#js-weekday-chart'),
    weekdayOptions
  );
  weekdayChart.render();
}

function plotMain(data) {
  const days =
    (new Date(data.end_time).getTime() - new Date(data.start_time).getTime()) /
    (1000 * 60 * 60 * 24);

  const mainData = [];
  const mainCategories = [];
  let chart = 'daily';

  if (days > 45) {
    chart = 'weekly';
  }

  if (days > 90) {
    chart = 'monthly';
  }

  if (days > 366) {
    chart = 'yearly';
  }

  data[chart].forEach((item) => {
    let label = item.name;
    if (chart === 'daily') {
      label = new Date(item.name).toLocaleDateString(navigator.language, {
        month: 'short',
        day: 'numeric',
      });
    }

    mainData.push(Math.floor(item.duration / 60000000000));
    mainCategories.push(label);
  });

  const mainOptions = getChartOptions(
    mainData,
    mainCategories,
    `${toTitleCase(chart)} totals`
  );

  const mainChart = new ApexCharts(
    document.getElementById('js-main-chart'),
    mainOptions
  );
  mainChart.render();
}

function plotHourly(data) {
  const hourlyData = [];
  const hourlyCategories = [];
  data.hourly.forEach((item) => {
    hourlyCategories.push(item.name);
    hourlyData.push(Math.floor(item.duration / 60000000000));
  });

  const hourlyOptions = getChartOptions(
    hourlyData,
    hourlyCategories,
    'Hourly totals'
  );
  hourlyOptions.chart.type = 'area';

  const hourlyChart = new ApexCharts(
    document.querySelector('#js-hourly-chart'),
    hourlyOptions
  );
  hourlyChart.render();
}

function plotTags(data) {
  const tagsData = [];
  const tagsCategories = [];
  data.tags.forEach((item) => {
    tagsCategories.push(item.name);
    tagsData.push(Math.floor(item.duration / 60000000000));
  });

  const tooltip = {
    y: {
      formatter: (value) => {
        return toHoursAndMinutes(value);
      },
    },
  };

  const tagOptions = {
    series: tagsData,
    chart: {
      height: 300,
      type: 'pie',
    },
    labels: tagsCategories,
    tooltip,
    title: {
      text: 'Tags',
      margin: 20,
      style: {
        fontSize: '24px',
      },
    },
  };

  const tagChart = new ApexCharts(
    document.querySelector('#js-tags-chart'),
    tagOptions
  );
  tagChart.render();
}

document.addEventListener('DOMContentLoaded', async () => {
  try {
    const pickerEl = document.getElementById('datepicker');
    const startDate = new Date(pickerEl.dataset.start);
    const endDate = new Date(pickerEl.dataset.end);

    const presetDates = getDateRanges();

    new easepick.create({
      element: pickerEl,
      css: ['/web/css/easepick_v1.2.1.css'],
      zIndex: 10,
      plugins: [RangePlugin, PresetPlugin, TimePlugin],
      RangePlugin: {
        startDate,
        endDate,
      },
      PresetPlugin: {
        customPreset: presetDates,
      },
      setup(picker) {
        picker.on('select', (e) => {
          const { start, end } = e.detail;
          window.location.href = `${
            window.location.pathname
          }?start_time=${formatDate(start)}&end_time=${formatDate(end)}`;
        });
      },
    });

    const body = document.getElementById('body');
    const data = JSON.parse(body.dataset.stats);

    plotSummary(data);
    plotMain(data);
    plotWeekday(data);
    plotHourly(data);
    plotTags(data);
  } catch (err) {
    console.log(err);
  }
});
