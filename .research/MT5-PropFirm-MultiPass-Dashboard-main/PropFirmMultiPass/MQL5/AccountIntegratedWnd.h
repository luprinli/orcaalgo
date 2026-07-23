
// #if !defined(AFX_ACCOUNTINTEGRATEDWND_H__1FE251B8_B64F_47C2_B695_4BFB32FC6E83__INCLUDED_)
// #define AFX_ACCOUNTINTEGRATEDWND_H__1FE251B8_B64F_47C2_B695_4BFB32FC6E83__INCLUDED_g

// #if _MSC_VER > 1000
// #pragma once
// #endif // _MSC_VER > 1000

// #include "Window.h"

// // �˻���Ϣ���ڣ���ʾһ���˻������µ��˻�
// class AccountIntegratedWnd : public SpwWindow
// {
// public:
//     AccountIntegratedWnd( CWnd * pParentWnd, LPCTSTR lpszWindowName, const RECT& rect );
//     virtual ~AccountIntegratedWnd();

//     // ������ʾ�˻���Ϣ
//     void SetAccountsInfo( AccountCate const & cate, AccountArray const & accounts );

// protected:
//     // ���´���������ض���
//     void RefreshAllCreate();
//     // ����
//     void MakeDraw();

//     winplus::WindowTimer m_timer1; // ��ʱ��

//     /* �˻���Ϣ */
//     AccountCate m_cate;
//     struct AccountContext
//     {
//         // �˻�
//         Account account;
//         // �˻��滭����
//         RectF accountRect;
//         // �˻��ֶ���ʾ����
//         RectF accountField0Rect;
//         RectF accountField1Rect;
//         // �Ƿ�ѡ��
//         //bool isSelected;
//         // �Ƿ���ʾ����
//         bool isPwdShown;
//         AccountContext()
//         {
//             this->_zeroInit();
//         }
//         AccountContext( Account const & account ) : account(account)
//         {
//             this->_zeroInit();
//         }
//         void _zeroInit()
//         {
//             using MyRect = decltype(accountRect);
//             ZeroMemory( &accountRect, sizeof(MyRect) );
//             ZeroMemory( &accountField0Rect, sizeof(MyRect) );
//             ZeroMemory( &accountField1Rect, sizeof(MyRect) );
//             //isSelected = false;
//             isPwdShown = false;
//         }
//     };
//     CArray<AccountContext> m_accounts;

//     /* ��ͼ���� */
//     winplus::SimplePointer<Bitmap> m_loadedBgImg; // ����ͼ
//     winplus::SimplePointer<winplus::Graphics> m_gCanvas; // ��ͼ����
//     winplus::MemImage m_memCanvas; // �ڴ滭��

//     CRect m_rcClient; // �ͻ�������
//     CPoint m_ptCurMouse; // ��ǰ���λ��
//     int m_radiusMouseCircle; // �����ЧԲ�ΰ뾶
//     RectF m_captionRect; // �������
//     RectF m_contentRect; // ���ݾ���

//     Gdiplus::Font m_primaryFont; // ������
//     StringFormat m_sfHVCenter; // �ַ�����������ʽ
//     Pen m_penBlack;
//     Pen m_penHalfWhite;
//     Pen m_penWhite;
//     SolidBrush m_brushHalfWhite;
//     SolidBrush m_brushBlack;
//     SolidBrush m_brushWhite;

//     // ClassWizard generated virtual function overrides
//     //{{AFX_VIRTUAL(AccountIntegratedWnd)
// protected:
//     virtual BOOL PreCreateWindow(CREATESTRUCT& cs);
//     //}}AFX_VIRTUAL

// protected:
//     //{{AFX_MSG(AccountIntegratedWnd)
//     afx_msg int OnCreate(LPCREATESTRUCT lpCreateStruct);
//     afx_msg LRESULT OnNcHitTest(CPoint point);
//     afx_msg BOOL OnEraseBkgnd(CDC* pDC);
//     afx_msg void OnPaint();
//     afx_msg void OnTimer(UINT_PTR nIDEvent);
//     afx_msg void OnMouseMove(UINT nFlags, CPoint point);
//     afx_msg void OnLButtonDown( UINT nFlags, CPoint point );
//     afx_msg void OnLButtonUp( UINT nFlags, CPoint point );
//     afx_msg void OnRButtonDown( UINT nFlags, CPoint point );
//     afx_msg void OnRButtonUp( UINT nFlags, CPoint point );
//     //}}AFX_MSG

//     DECLARE_MESSAGE_MAP()
// public:
// };

// /////////////////////////////////////////////////////////////////////////////

// //{{AFX_INSERT_LOCATION}}
// // Microsoft Visual C++ will insert additional declarations immediately before the previous line.

// #endif // !defined(AFX_ACCOUNTINTEGRATEDWND_H__1FE251B8_B64F_47C2_B695_4BFB32FC6E83__INCLUDED_)
