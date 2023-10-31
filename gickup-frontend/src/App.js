import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import { Container, AppBar, Toolbar, Typography, Button, Box } from '@mui/material';
import StepWizard from 'react-step-wizard';
import Step1 from './description'; 
import Step2 from './source';
import Step3 from './destination';
import Step4 from './scheduler';
import Step5 from './summary';
import SourceConfig from './source';
import DestinationConfig from './destination';
import SchedulerConfig from './scheduler';
import Stats from './stats';
import Jobs from './joblist';

function App() {
  return (
    <Router>
      <Box sx={{ flexGrow: 1, bgcolor: '#f0f0f0', minHeight: '100vh' }}>
        <AppBar position="static" sx={{ bgcolor: '#4caf50' }}>
          <Toolbar>
            <Typography variant="h6" component={Link} to="/" style={{ textDecoration: 'none', color: 'white' }}>
              Source Code Backup
            </Typography>
            <nav style={{ marginLeft: '20px' }}>
            <Button component={Link} to="/wizard" variant="contained" style={{  marginRight: '10px' }}>
                Create Backup Job
              </Button>
              <Button component={Link} to="" variant="contained" style={{ marginRight: '10px' }}>
                Backup History
              </Button>
              <Button component={Link} to="/joblist" variant="contained"style={{ marginRight: '10px' }}>
                Backup Jobs
              </Button>
              <Button component={Link} to="/stats" variant="contained">
                Backup Statistics
              </Button>
            </nav>
          </Toolbar>
        </AppBar>

        <Container
          maxWidth="md"
          style={{ marginTop: '20px' }}
          sx={{ backgroundColor: '#f5f5f5', padding: '20px', minHeight: '100vh' }}
        >
          <Routes>
            <Route path="/source" element={<SourceConfig />} />
            <Route path="/destination" element={<DestinationConfig />} />
            <Route path="/scheduler" element={<SchedulerConfig />} />
            <Route path="/stats" element={<Stats />} />
            <Route path="/joblist" element={<Jobs />} />
            <Route
              path="/wizard"
              element={
                <StepWizard isLazyMount>
                  <Step1 />
                  <Step2 />
                  <Step3 />
                  <Step4 />
                  <Step5 />
                </StepWizard>
              }
            />
          </Routes>
        </Container>
      </Box>
    </Router>
  );
}

export default App;
