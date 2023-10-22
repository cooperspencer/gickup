import React, { useState } from 'react';
import { Routes, Route, useNavigate } from 'react-router-dom';
import NameDescriptionStep from './description';
import SourceStep from './source';
import DestinationStep from './destination';
import SchedulerStep from './scheduler';
import SummaryStep from './summary';

const Wizard = () => {
  const navigate = useNavigate();
  const [sourceConfig, setSourceConfig] = useState({});

  const handleSourceNext = (config) => {
    setSourceConfig(config);
    navigate('/destination');
  };

  const handleDestinationNext = (config) => {
    setSourceConfig(config);
    navigate('/scheduler');
  };

  const handleSchedulerNext = () => {
    navigate('/summary');
  };

  return (
    <Routes>
      <Route path="/description" element={<NameDescriptionStep />} />
      <Route path="/source" element={<SourceStep onNext={handleSourceNext} onPrevious={() => navigate('/description')} />} />
      <Route path="/destination" element={<DestinationStep onNext={handleDestinationNext} onPrevious={() => navigate('/source')} />} />
      <Route path="/scheduler" element={<SchedulerStep onNext={handleSchedulerNext} onPrevious={() => navigate('/destination')} />} />
      <Route path="/summary" element={<SummaryStep />} />
    </Routes>
  );
};

export default Wizard;
