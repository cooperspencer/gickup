import React, { useEffect, useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
} from '@mui/material';

const History = () => {
  const [backupHistory, setBackupHistory] = useState([]);

  useEffect(() => {
    fetch('http://localhost:5000/api/fetchLogFile')
      .then((response) => response.text())
      .then((data) => {
        // Split the log data into lines
        const lines = data.split('\n');
        
        const formattedData = lines.map((line) => {
          const type = line.includes('ERR') ? 'ERR' : 'INF';
          const [timestamp, message] = line.split(type).map((item) => item.trim());
          return { type, timestamp, message };
        });
        setBackupHistory(formattedData);
      })
      .catch((error) => {
        console.error('Error fetching backup history:', error);
      });
  }, []);

  return (
    <div style={{ padding: '20px' }}>
      <h1>Backup History</h1>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Type</TableCell>
              <TableCell>Timestamp</TableCell>
              <TableCell>Message</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {backupHistory.map((row, index) => (
              <TableRow key={index}>
                <TableCell>{row.type}</TableCell>
                <TableCell dangerouslySetInnerHTML={{ __html: row.timestamp }} />
                <TableCell dangerouslySetInnerHTML={{ __html: row.message }} />
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  );
};

export default History;
