import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import { faFile } from '@fortawesome/free-solid-svg-icons/faFile';
import FileCode from './file-code.svg';
import Folder from './folder-plus.svg';

const FileTree = ({ data, onFolderClick, onFileClick, expandedFolders, setExpandedFolders, currentPath }) => {
  const handleFolderClick = (folderName) => {
    if (!expandedFolders[folderName]) {
      const newPath = currentPath ? `${currentPath}/${folderName}` : folderName;
      onFolderClick(newPath);
    }

    const newExpandedFolders = { ...expandedFolders };
    newExpandedFolders[folderName] = !expandedFolders[folderName];
    setExpandedFolders(newExpandedFolders);
  };

  return (
    <ul>
      {data.items.map((item, index) => (
        <li key={index}>
          {item.type === 'folder' ? (
            <>
              <img src={Folder} alt="Folder Icon" style={{ marginRight: '5px', width: '25px', height: '25px' }} />
              <span onClick={() => handleFolderClick(item.name)}>
                {expandedFolders[item.name] ? '[-] ' : '[+] '}{item.name}
              </span>
              {expandedFolders[item.name] && (
                <FileTree
                  data={item}
                  onFolderClick={onFolderClick}
                  expandedFolders={expandedFolders}
                  setExpandedFolders={setExpandedFolders}
                  currentPath={currentPath}
                  onFileClick={onFileClick} 
                />
              )}
            </>
          ) : (
            
            <div style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }} onClick={() => onFileClick(item.name)}>
              <img src={FileCode} alt="File Icon" style={{ marginRight: '5px', width: '25px', height: '25px' }} />
              <span>{item.name}</span>
            </div>
          )}
        </li>
      ))}
    </ul>
  );
};

const FileExplorer = () => {
  const [currentPath, setCurrentPath] = useState('');
  const [directoryContent, setDirectoryContent] = useState(null);
  const [loading, setLoading] = useState(true);
  const [expandedFolders, setExpandedFolders] = useState({});
  const [fileContent, setFileContent] = useState('');

  const fetchData = useCallback(async (path) => {
    setLoading(true);
    try {
      const response = await axios.get('http://localhost:5000/api/files', { params: { path } });
      const transformedData = transformData(response.data);
      setDirectoryContent(transformedData);
    } catch (error) {
      console.error('Error fetching directory content:', error);
    } finally {
      setLoading(false);
    }
  }, []); 

  const transformData = (response) => {
    const transformFolder = (folder) => ({
      name: folder.name,
      type: 'folder',
      items: Object.entries(folder.folders).map(([key, value]) => transformFolder({ name: key, ...value }))
        .concat(folder.files.map((fileName) => ({ name: fileName, type: 'file' }))),
    });

    const rootFolder = transformFolder({
      name: 'Root',
      folders: response.folders,
      files: response.files,
    });

    return rootFolder;
  };

  const BACKUP_ROOT_FOLDER = '/backups/'; 

  const handleFileClick = async (filePath) => {
    const fullPath = BACKUP_ROOT_FOLDER + currentPath + '/' + filePath;
    console.log('Clicked File Path:', fullPath); 
    try {
      const response = await axios.get('http://localhost:5000/api/files/content', { params: { path: fullPath } });
      setFileContent(response.data);
    } catch (error) {
      console.error('Error fetching file content:', error);
    }
  };

  useEffect(() => {
    fetchData('');
  }, [fetchData]);

  if (loading) {
    return <div>Loading...</div>;
  }

  if (!directoryContent) {
    return <div>Error loading directory content.</div>;
  }

  return (
    <div style={{ display: 'flex' }}>
      <div style={{ width: '50%', paddingRight: '20px' }}>
        <h2>Backup File Browser</h2>
        <div style={{ height: 500 }}>
          <FileTree
            data={directoryContent}
            onFolderClick={setCurrentPath}
            onFileClick={handleFileClick} // Pass the file click handler
            expandedFolders={expandedFolders}
            setExpandedFolders={setExpandedFolders}
            currentPath={currentPath}
          />
        </div>
      </div>
      <div style={{ width: '50%' }}>
        <h2>File Preview</h2>
        <pre>{fileContent}</pre>
      </div>
    </div>
  );
};


export default FileExplorer;
