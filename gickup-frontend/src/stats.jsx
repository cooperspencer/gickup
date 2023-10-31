import React, { useEffect, useState } from 'react';
import axios from 'axios';

function Stats() {
  const [backupStatistics, setBackupStatistics] = useState(null);

  useEffect(() => {
    // Fetch data from the API endpoint when the component mounts
    axios
      .get('http://localhost:5000/api/backupStatistics')
      .then(response => {
        setBackupStatistics(response.data.backupStatistics);
      })
      .catch(error => {
        console.error('Error fetching backup statistics:', error);
      });
  }, []); // Empty dependency array ensures this effect runs once after the initial render

  return (
    <div>
      <h2>Backup Statistics</h2>
      {backupStatistics ? (
        <div>
          <p>Successful Runs: {backupStatistics.successfulRuns}</p>
          <p>Total Duration: {backupStatistics.totalDuration}</p>
        </div>
      ) : (
        <p>Loading data...</p>
      )}
    </div>
  );
}

export default Stats;
