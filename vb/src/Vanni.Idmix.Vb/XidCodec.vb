Imports System.IO

Namespace Vanni.Idmix

''' <summary>XID v1.1 二进制层编解码。</summary>
Public NotInheritable Class XidCodec
    Private Shared ReadOnly SwBytes As Integer() = {1, 2, 4, 8}
    Private Shared ReadOnly EmbeddedOType As Integer(,) = {
        {TypedValue.OTypeUint8, TypedValue.OTypeUint16, TypedValue.OTypeUint32, TypedValue.OTypeUint64},
        {TypedValue.OTypeInt8, TypedValue.OTypeInt16, TypedValue.OTypeInt32, TypedValue.OTypeInt64}
    }

    Private Sub New()
    End Sub

    Public Shared Function EncodeBinary(m As IdMix, typed As IList(Of TypedValue), variantId As Integer) As Byte()
        Using objects As New MemoryStream(typed.Count * 9)
            For Each tv In typed
            Dim obj = EncodeObject(tv)
            objects.Write(obj, 0, obj.Length)
        Next
            Dim objBytes = objects.ToArray()
            Dim mask = CByte((variantId * &H9D + &H37) And &HFF)
            For i = 0 To objBytes.Length - 1
                objBytes(i) = objBytes(i) Xor mask
            Next
            Dim count = typed.Count
            Dim header = (variantId << m.VariantShift) Or (count << m.CountShift)
            Dim data(1 + objBytes.Length) As Byte
            data(0) = CByte(header And &HFF)
            data(1) = CByte((header >> 8) And &HFF)
            Buffer.BlockCopy(objBytes, 0, data, 2, objBytes.Length)
            Dim xorSum = 0
            For Each b In data
                xorSum = xorSum Xor b
            Next
            header = header Or (xorSum And m.CheckMask)
            data(0) = CByte(header And &HFF)
            data(1) = CByte((header >> 8) And &HFF)
            Return data
        End Using
    End Function

    Public Shared Function DecodeBinary(m As IdMix, data As Byte()) As List(Of TypedValue)
        If data.Length < 2 Then Throw New ArgumentException("invalid data: too short")
        Dim header = data(0) Or (data(1) << 8)
        Dim check = header And m.CheckMask
        Dim count = (header And m.CountMask) >> m.CountShift
        Dim variantId = (header And m.VariantMask) >> m.VariantShift
        If variantId >= m.MaxVariants Then Throw New ArgumentException($"invalid variant_id {variantId}")
        If count > m.MaxObjects Then Throw New ArgumentException($"invalid count {count}")
        Dim verify = CType(data.Clone(), Byte())
        verify(0) = CByte(verify(0) And Not m.CheckMask)
        Dim xorSum = 0
        For Each b In verify
            xorSum = xorSum Xor b
        Next
        If (xorSum And m.CheckMask) <> check Then Throw New ArgumentException("checksum mismatch")
        Dim objects(data.Length - 3) As Byte
        Buffer.BlockCopy(data, 2, objects, 0, objects.Length)
        Dim mask = CByte((variantId * &H9D + &H37) And &HFF)
        For i = 0 To objects.Length - 1
            objects(i) = objects(i) Xor mask
        Next
        Dim result As New List(Of TypedValue)()
        Dim pos = 0
        For i = 0 To count - 1
            If pos >= objects.Length Then Throw New ArgumentException("premature end of data")
            Dim dr = DecodeObject(objects, pos)
            result.Add(dr.Tv)
            pos += dr.Consumed
        Next
        If pos <> objects.Length Then Throw New ArgumentException("extra bytes after data objects")
        Return result
    End Function

    Private Shared Function EncodeObject(tv As TypedValue) As Byte()
        ValidateRange(tv.OType, tv.Val)
        Dim embedded = TryEmbeddedHead(tv.OType, tv.Val)
        If embedded.HasValue Then Return {embedded.Value}
        Dim magNeg = MagnitudeFromTyped(tv.OType, tv.Val)
        Dim mag = magNeg.Mag
        Dim neg = magNeg.Neg
        Dim sw = SwFromMagnitude(mag)
        Dim payload = UintToLeBytes(mag, SwBytes(sw))
        Dim head = &H80 Or (sw << 4) Or tv.OType
        If neg Then head = head Or (1 << 6)
        Dim outBuf(1 + payload.Length - 1) As Byte
        outBuf(0) = CByte(head)
        Buffer.BlockCopy(payload, 0, outBuf, 1, payload.Length)
        Return outBuf
    End Function

    Private Structure DecodeResult
        Public Tv As TypedValue
        Public Consumed As Integer
    End Structure

    Private Shared Function DecodeObject(data As Byte(), offset As Integer) As DecodeResult
        If offset >= data.Length Then Throw New ArgumentException("truncated object header")
        Dim head = data(offset)
        If (head And &H80) = 0 Then
            Dim sign = (head >> 6) And 1
            Dim wb = (head >> 4) And &H03
            Dim v = head And &H0F
            Dim otype = EmbeddedOType(sign, wb)
            Dim val = If(sign = 0, CLng(v), -v - 1L)
            Return New DecodeResult With {.Tv = New TypedValue(otype, val), .Consumed = 1}
        End If
        Dim sw = (head >> 4) And &H03
        Dim otype2 = head And &H0F
        If otype2 > TypedValue.OTypeInt64 Then Throw New ArgumentException($"invalid otype {otype2}")
        Dim numBytes = SwBytes(sw)
        If data.Length < offset + 1 + numBytes Then Throw New ArgumentException("truncated object payload")
        Dim mag As Long = 0
        For i = 0 To numBytes - 1
            mag = mag Or (CLng(data(offset + 1 + i) And &HFF) << (8 * i))
        Next
        Dim neg = ((head >> 6) And 1) <> 0
        Dim val2 = ValueFromMagnitude(mag, neg)
        ValidateRange(otype2, val2)
        Return New DecodeResult With {.Tv = New TypedValue(otype2, val2), .Consumed = 1 + numBytes}
    End Function

    Private Shared Function IsUnsigned(otype As Integer) As Boolean
        Return otype <= TypedValue.OTypeUint64
    End Function

    Private Shared Function IsSigned(otype As Integer) As Boolean
        Return otype >= TypedValue.OTypeInt8
    End Function

    Private Shared Function WidthBits(otype As Integer) As Integer
        Select Case otype
            Case TypedValue.OTypeUint8, TypedValue.OTypeInt8 : Return 0
            Case TypedValue.OTypeUint16, TypedValue.OTypeInt16 : Return 1
            Case TypedValue.OTypeUint32, TypedValue.OTypeInt32 : Return 2
            Case Else : Return 3
        End Select
    End Function

    Private Structure MagNeg
        Public Mag As Long
        Public Neg As Boolean
    End Structure

    Private Shared Function MagnitudeFromTyped(otype As Integer, val As Long) As MagNeg
        If IsUnsigned(otype) Then Return New MagNeg With {.Mag = val, .Neg = False}
        If val < 0 Then Return New MagNeg With {.Mag = -val, .Neg = True}
        Return New MagNeg With {.Mag = val, .Neg = False}
    End Function

    Private Shared Function SwFromMagnitude(mag As Long) As Integer
        If CULng(mag) < 256 Then Return 0
        If CULng(mag) < 65536 Then Return 1
        If CULng(mag) < 4294967296UL Then Return 2
        Return 3
    End Function

    Private Shared Function TryEmbeddedHead(otype As Integer, val As Long) As Byte?
        Dim magNeg = MagnitudeFromTyped(otype, val)
        If CULng(magNeg.Mag) >= 17 Then Return Nothing
        Dim wb = WidthBits(otype)
        If CULng(magNeg.Mag) = 16 Then
            If magNeg.Neg Then Return CByte((1 << 6) Or (wb << 4) Or 15)
            Return Nothing
        End If
        If magNeg.Neg Then Return CByte((1 << 6) Or (wb << 4) Or CInt(magNeg.Mag - 1))
        Return CByte((wb << 4) Or CInt(magNeg.Mag))
    End Function

    Private Shared Function ValueFromMagnitude(mag As Long, neg As Boolean) As Long
        If Not neg Then Return mag
        If mag = 1L << 63 Then Return Long.MinValue
        Return -mag
    End Function

    Private Shared Function UintToLeBytes(v As Long, size As Integer) As Byte()
        Dim buf(size - 1) As Byte
        Dim u As ULong = CULng(v)
        For i = 0 To size - 1
            buf(i) = CByte((u >> (8 * i)) And &HFF)
        Next
        Return buf
    End Function

    Private Shared Sub ValidateRange(otype As Integer, val As Long)
        Dim ok As Boolean
        Select Case otype
            Case TypedValue.OTypeUint8 : ok = val >= 0 AndAlso val <= &HFF
            Case TypedValue.OTypeUint16 : ok = val >= 0 AndAlso val <= &HFFFF
            Case TypedValue.OTypeUint32 : ok = val >= 0 AndAlso val <= &HFFFFFFFFL
            Case TypedValue.OTypeUint64 : ok = True
            Case TypedValue.OTypeInt8 : ok = val >= -128 AndAlso val <= 127
            Case TypedValue.OTypeInt16 : ok = val >= -32768 AndAlso val <= 32767
            Case TypedValue.OTypeInt32 : ok = val >= Integer.MinValue AndAlso val <= Integer.MaxValue
            Case TypedValue.OTypeInt64 : ok = True
            Case Else : ok = False
        End Select
        If Not ok Then Throw New ArgumentException($"value {val} out of range for otype {otype}")
    End Sub
End Class

End Namespace
