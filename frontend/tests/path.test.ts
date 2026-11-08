import {test, expect} from 'vitest';
import {
    includesPath,
    pathBasename,
    pathDirname,
    pathDisplayDir,
    pathKey,
    samePath,
} from '../src/lib/utils/path';

test('path helpers handle POSIX and Windows separators', () => {
    expect(pathBasename('/home/lain/Pictures/wall.jpg')).toBe('wall.jpg');
    expect(pathBasename('C:\\Users\\enrell\\Pictures\\wall.jpg')).toBe(
        'wall.jpg'
    );
    expect(pathBasename('wall.jpg')).toBe('wall.jpg');

    expect(pathDirname('/home/lain/Pictures/wall.jpg')).toBe(
        '/home/lain/Pictures'
    );
    expect(pathDirname('C:\\Users\\enrell\\Pictures\\wall.jpg')).toBe(
        'C:\\Users\\enrell\\Pictures'
    );
    expect(pathDirname('C:\\Users\\enrell\\Pictures\\')).toBe(
        'C:\\Users\\enrell\\Pictures'
    );
    expect(pathDirname('/wall.jpg')).toBe('/');
    expect(pathDirname('wall.jpg')).toBe('');
});

test('Windows local paths compare case-insensitively', () => {
    expect(pathKey('C:\\Users\\Enrell\\wall.jpg')).toBe(
        'c:\\users\\enrell\\wall.jpg'
    );
    expect(
        samePath('C:\\Users\\Enrell\\wall.jpg', 'c:\\users\\enrell\\WALL.JPG')
    ).toBe(true);
    expect(
        includesPath(
            ['C:\\Users\\Enrell\\wall.jpg'],
            'c:\\users\\enrell\\WALL.JPG'
        )
    ).toBe(true);
    // POSIX paths and URLs keep their case.
    expect(samePath('/home/Enrell/wall.jpg', '/home/enrell/wall.jpg')).toBe(
        false
    );
    expect(
        samePath('https://example.com/Wall.jpg', 'https://example.com/wall.jpg')
    ).toBe(false);
});

test('the home prefix is shortened on every platform', () => {
    expect(pathDisplayDir('/home/lain/Pictures/wall.jpg')).toBe('~/Pictures');
    expect(pathDisplayDir('/Users/lain/Pictures/wall.jpg')).toBe('~/Pictures');
    expect(pathDisplayDir('C:\\Users\\enrell\\Pictures\\wall.jpg')).toBe(
        '~\\Pictures'
    );
    expect(pathDisplayDir('D:\\Wallpapers\\wall.jpg')).toBe('D:\\Wallpapers');
});
