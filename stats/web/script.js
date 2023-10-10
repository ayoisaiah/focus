import ApexCharts from 'apexcharts';

function toTitleCase(str) {
  return str.replace(/\w\S*/g, function (word) {
    return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase();
  });
}

function toHoursAndMinutes(totalMinutes) {
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;

  return `${hours} h${minutes > 0 ? ` ${minutes} m` : ''}`;
}

function plotSummary(data) {
  document.querySelector('#js-total-time').textContent = toHoursAndMinutes(
    Math.ceil(data.totals.duration / 60000000000)
  );
  document.querySelector('#js-top-tag').textContent = `${
    data.tags[0].name
  } (${toHoursAndMinutes(Math.ceil(data.tags[0].duration / 60000000000))})`;
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

  const tagOptions = {
    series: tagsData,
    chart: {
      height: 300,
      type: 'pie',
    },
    labels: tagsCategories,
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
    const main = document.getElementById('main');
    const data = JSON.parse(main.dataset.stats);

    plotSummary(data);
    plotMain(data);
    plotWeekday(data);
    plotHourly(data);
    plotTags(data);
  } catch (err) {
    console.log(err);
  }
});
