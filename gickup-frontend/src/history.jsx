import React, { useState, useEffect } from 'react';
import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Paper } from '@mui/material';

const BackupHistory = ({ backupHistory }) => {
  return (
    <TableContainer component={Paper} style={{ marginTop: '20px' }}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Date</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Details</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {backupHistory.map((backup, index) => (
            <TableRow key={index}>
              <TableCell>{backup.date}</TableCell>
              <TableCell>{backup.status}</TableCell>
              <TableCell>{backup.details}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

const History = () => {
  const [backupHistory, setBackupHistory] = useState([]);

  useEffect(() => {
    // Fetch backup history data from the API endpoint
    fetch('http://localhost:5000/api/backupHistory') // Replace with your actual API endpoint
      .then((response) => response.json())
      .then((data) => {
        setBackupHistory(data); // Assuming the API response is an array of backup history objects
      })
      .catch((error) => {
        console.error('Error fetching backup history:', error);
      });
  }, []);

  return (
    <div style={{ padding: '20px' }}>
      <h1>Backup History</h1>
      <BackupHistory backupHistory={backupHistory} />
    </div>
  );
};

export default History;
