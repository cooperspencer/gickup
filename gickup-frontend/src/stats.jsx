import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend } from 'recharts';

function Stats() {
  const [backupData, setBackupData] = useState([]);

  useEffect(() => {
    axios
      .get('http://localhost:5000/api/backupStatistics')
      .then(response => {
        console.log('API output', response.data); // Log the API response

        const { successfulRuns, totalDuration } = response.data.backupData;

        // Ensure individualDurations is an array before setting the state
        if (Array.isArray(totalDuration.individualDurations)) {
          setBackupData([
            { name: 'Total', successfulRuns, duration: totalDuration.total },
            ...totalDuration.individualDurations.map((duration, index) => ({ name: `Run ${index + 1}`, duration }))
          ]);
        } else {
          console.error('Invalid individualDurations format received from the server:', totalDuration.individualDurations);
        }
      })
      .catch(error => {
        console.error('Error fetching backup statistics:', error);
      });
  }, []);

  if (!Array.isArray(backupData) || backupData.length === 0) {
    return <p>Loading data...</p>;
  }

  return (
    <div>
      <h2>Backup Statistics</h2>
      <BarChart width={800} height={400} data={backupData}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="name" tick={{ fontSize: 14, fill: '#333', fontWeight: 'bold' }} />
        <YAxis tick={{ fontSize: 14, fill: '#333', fontWeight: 'bold' }} />
        <Tooltip />
        <Legend wrapperStyle={{ fontSize: 16, color: '#333', fontWeight: 'bold' }} />
        <Bar dataKey="successfulRuns" fill="#007bff" stroke="#007bff" strokeWidth={1} name="Successful Runs" />
        <Bar dataKey="duration" fill="#28a745" stroke="#28a745" strokeWidth={1} name="Backup Durations" />
      </BarChart>
    </div>
  );
}

export default Stats;
